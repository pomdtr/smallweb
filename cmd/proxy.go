package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

const (
	userNameContextKey = "username"
	userIDContextKey   = "userid"
)

type Message[T any] struct {
	ID   string `json:"id"`
	Data T      `json:"data"`
}

type SetUserNameRequest struct {
	Username string
}

func NewCmdProxy() *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "Start a smallweb proxy",
		RunE: func(cmd *cobra.Command, args []string) error {

			db, err := NewDB("smallweb.db")
			if err != nil {
				log.Fatalf("failed to open database: %v", err)
			}

			requestForwarder := NewRequestForwarder()

			httpServer := http.Server{
				Addr: ":8000",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					username := strings.Split(r.Host, "-")[0]

					var headers [][]string
					for key, values := range r.Header {
						headers = append(headers, []string{key, values[0]})
					}

					body, err := io.ReadAll(r.Body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					req, err := requestForwarder.forward(username, SerializedRequest{
						Url:     fmt.Sprintf("https://%s%s", r.Host, r.URL.Path),
						Method:  r.Method,
						Headers: headers,
						Body:    body,
					})
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for _, header := range req.Headers {
						w.Header().Set(header[0], header[1])
					}

					w.WriteHeader(req.Status)
					w.Write(req.Body)
				}),
			}

			sshServer := ssh.Server{
				Addr: ":2222",
				PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
					keyText, err := keyText(key)
					if err != nil {
						return false
					}
					user, err := db.UserForKey(keyText, true)
					if err != nil {
						return false
					}

					ctx.SetValue(userNameContextKey, user.Name)
					ctx.SetValue(userIDContextKey, user.PublicID)

					return true
				},
				RequestHandlers: map[string]ssh.RequestHandler{
					"get-username": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
						username := ctx.Value(userNameContextKey).(string)
						return true, []byte(username)
					},
					"set-username": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
						userID := ctx.Value(userIDContextKey).(string)
						var payload SetUserNameRequest
						if err := gossh.Unmarshal(req.Payload, &payload); err != nil {
							return false, nil
						}

						if _, err := db.SetUserName(userID, payload.Username); err != nil {
							return false, nil
						}

						ctx.SetValue(userNameContextKey, payload.Username)
						return true, nil
					},
				},
				ChannelHandlers: map[string]ssh.ChannelHandler{
					"smallweb": func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
						username := ctx.Value(userNameContextKey).(string)
						if username == "" {
							newChan.Reject(gossh.Prohibited, "no username in context")
						}
						channel, reqs, err := newChan.Accept()
						if err != nil {
							log.Fatalf("accept failed: %v", err)
						}

						go gossh.DiscardRequests(reqs)

						if err := requestForwarder.watchChannel(username, channel); err != nil {
							slog.Error("failed to watch channel", "error", err)
						}
					},
				},
			}

			slog.Info("starting servers")
			go sshServer.ListenAndServe()
			go httpServer.ListenAndServe()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			slog.Info("shutting down servers")
			httpServer.Close()
			sshServer.Close()
			return nil
		},
	}
}

type RequestForwarder struct {
	sync.Mutex
	channels map[string]gossh.Channel
	outputs  map[string]chan SerializedResponse
}

func NewRequestForwarder() *RequestForwarder {
	return &RequestForwarder{
		channels: make(map[string]gossh.Channel),
		outputs:  make(map[string]chan SerializedResponse),
	}
}

func (rf *RequestForwarder) forward(to string, req SerializedRequest) (SerializedResponse, error) {
	channel, ok := rf.channels[to]
	if !ok {
		return SerializedResponse{}, fmt.Errorf("no channel for user %s", to)
	}

	requestID := uuid.New().String()
	msg := Message[SerializedRequest]{
		ID:   requestID,
		Data: req,
	}

	rf.Lock()
	output := make(chan SerializedResponse)
	rf.outputs[requestID] = output
	rf.Unlock()

	defer func() {
		rf.Lock()
		delete(rf.outputs, requestID)
		close(output)
		rf.Unlock()
	}()

	encoder := json.NewEncoder(channel)
	if err := encoder.Encode(msg); err != nil {
		return SerializedResponse{}, fmt.Errorf("failed to encode message: %w", err)
	}

	resp := <-output
	rf.Lock()
	delete(rf.outputs, requestID)
	rf.Unlock()

	return resp, nil
}

func (rf *RequestForwarder) watchChannel(username string, channel gossh.Channel) error {
	rf.Lock()
	rf.channels[username] = channel
	rf.Unlock()

	defer func() {
		rf.Lock()
		delete(rf.channels, username)
		rf.Unlock()
	}()

	decoder := json.NewDecoder(channel)
	for {
		var msg Message[SerializedResponse]
		if err := decoder.Decode(&msg); err != nil {
			return err
		}

		output, ok := rf.outputs[msg.ID]
		if !ok {
			slog.Warn("no output channel", "message", msg.ID)
			continue
		}
		output <- msg.Data
	}
}

// keyText is the base64 encoded public key for the glider.Session.
func keyText(publicKey gossh.PublicKey) (string, error) {
	kb := base64.StdEncoding.EncodeToString(publicKey.Marshal())
	return fmt.Sprintf("%s %s", publicKey.Type(), kb), nil
}
