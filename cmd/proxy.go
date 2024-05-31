package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

const (
	userNameContextKey = "username"
	userIDContextKey   = "userid"
)

type ForwardPayload struct {
	Username string
}

func NewCmdProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Start a smallweb proxy",
		RunE: func(cmd *cobra.Command, args []string) error {

			db, err := NewDB("smallweb.db")
			if err != nil {
				log.Fatalf("failed to open database: %v", err)
			}

			forwarder := Forwarder{}
			httpPort, _ := cmd.Flags().GetInt("http-port")
			httpServer := http.Server{
				Addr: fmt.Sprintf(":%d", httpPort),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					req, err := serializeRequest(r)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					resp, err := forwarder.Forward(req)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for _, header := range resp.Headers {
						w.Header().Set(header[0], header[1])
					}

					w.WriteHeader(resp.Code)
					w.Write(resp.Body)
				}),
			}

			sshPort, _ := cmd.Flags().GetInt("ssh-port")
			sshServer := ssh.Server{
				Addr: fmt.Sprintf(":%d", sshPort),
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
					// "get-username": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
					// 	username := ctx.Value(userNameContextKey).(string)
					// 	return true, []byte(username)
					// },
					// "set-username": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
					// userID := ctx.Value(userIDContextKey).(string)
					// var payload SetUserNameRequest
					// if err := gossh.Unmarshal(req.Payload, &payload); err != nil {
					// 	return false, nil
					// }

					// if _, err := db.SetUserName(userID, payload.Username); err != nil {
					// 	return false, nil
					// }

					// ctx.SetValue(userNameContextKey, payload.Username)
					// return true, nil
					// },
					"smallweb-forward": forwarder.HandleSSHRequest,
				},
			}

			slog.Info("starting servers")
			go sshServer.ListenAndServe()
			go httpServer.ListenAndServe()
			go forwarder.KeepAlive()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			slog.Info("shutting down servers")
			httpServer.Close()
			sshServer.Close()
			return nil
		},
	}

	cmd.Flags().Int("ssh-port", 2222, "port for the ssh server")
	cmd.Flags().Int("http-port", 8000, "port for the http server")

	return cmd
}

// Forwarder can be enabled by creating a Forwarder and
// adding the HandleSSHRequest callback to the server's RequestHandlers under
// tcpip-forward and cancel-tcpip-forward.
type Forwarder struct {
	conns map[string]*gossh.ServerConn
	sync.Mutex
}

func (h *Forwarder) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	var payload ForwardPayload
	if err := gossh.Unmarshal(req.Payload, &payload); err != nil {
		return false, nil
	}

	h.Lock()
	if h.conns == nil {
		h.conns = make(map[string]*gossh.ServerConn)
	}
	h.Unlock()
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
	switch req.Type {
	case "smallweb-forward":
		h.Lock()
		h.conns[payload.Username] = conn
		h.Unlock()
		return true, nil
	case "cancel-smallweb-forward":
		h.Lock()
		delete(h.conns, payload.Username)
		h.Unlock()
		return true, nil
	default:
		return false, nil
	}
}

func (me *Forwarder) Forward(req *Request) (*Response, error) {
	username, err := req.Username()
	if err != nil {
		return nil, err
	}

	conn, ok := me.conns[username]
	if !ok {
		return nil, fmt.Errorf("no connection found")
	}

	ch, reqs, err := conn.OpenChannel("forwarded-smallweb", nil)
	if err != nil {
		return nil, fmt.Errorf("could not open channel: %v", err)
	}

	go gossh.DiscardRequests(reqs)

	encoder := json.NewEncoder(ch)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("could not encode request: %v", err)
	}

	decoder := json.NewDecoder(ch)
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("could not decode response: %v", err)
	}

	return &resp, nil
}

func (me *Forwarder) KeepAlive() {
	ticker := time.NewTicker(60 * time.Second)
	for {
		<-ticker.C
		for username, conn := range me.conns {
			_, _, err := conn.SendRequest("keepalive", true, nil)
			if err != nil {
				log.Printf("could not send keepalive: %v", err)
				me.Lock()
				delete(me.conns, username)
				me.Unlock()
			}
		}
	}
}

// keyText is the base64 encoded public key for the glider.Session.
func keyText(publicKey gossh.PublicKey) (string, error) {
	kb := base64.StdEncoding.EncodeToString(publicKey.Marshal())
	return fmt.Sprintf("%s %s", publicKey.Type(), kb), nil
}
