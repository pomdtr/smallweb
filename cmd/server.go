package cmd

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/caarlos0/env/v6"
	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

type LoginParams struct {
	Email string
}

type SignupParams struct {
	Email    string
	Username string
}

type VerifyEmailParams struct {
	Code string
}

type UserResponse struct {
	ID    string
	Name  string
	Email string
}

type ServerConfig struct {
	Host             string `env:"SMALLWEB_HOST" envDefault:"smallweb.run"`
	SSHPort          int    `env:"SMALLWEB_SSH_PORT" envDefault:"2222"`
	TursoDatabaseURL string `env:"TURSO_DATABASE_URL"`
	TursoAuthToken   string `env:"TURSO_AUTH_TOKEN"`
	ValTownToken     string `env:"VALTOWN_TOKEN"`
	Debug            bool   `env:"SMALLWEB_DEBUG" envDefault:"false"`
}

func ServerConfigFromEnv() (*ServerConfig, error) {
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func NewCmdServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "server",
		Short:  "Start a smallweb server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := ServerConfigFromEnv()
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			db, err := NewTursoDB(fmt.Sprintf("%s?authToken=%s", config.TursoDatabaseURL, config.TursoAuthToken))
			if err != nil {
				log.Fatalf("failed to open database: %v", err)
			}

			valtownClient := NewValTownClient(config.ValTownToken)

			forwarder := Forwarder{
				db:       db,
				forwards: make(map[string]int),
			}

			httpPort, _ := cmd.Flags().GetInt("http-port")
			httpServer := http.Server{
				Addr: fmt.Sprintf(":%d", httpPort),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					subdomain := strings.Split(r.Host, ".")[0]
					parts := strings.Split(subdomain, "-")
					user := parts[len(parts)-1]

					port, ok := forwarder.forwards[user]
					if !ok {
						http.Error(w, fmt.Sprintf("User %s not found", user), http.StatusNotFound)
						return
					}

					req, err := http.NewRequest(r.Method, fmt.Sprintf("http://127.0.0.1:%d%s", port, r.URL.String()), r.Body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for k, v := range r.Header {
						req.Header[k] = v
					}
					app := strings.Join(parts[:len(parts)-1], "-")
					req.Header.Add("X-Smallweb-App", app)

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					defer resp.Body.Close()

					for k, v := range resp.Header {
						for _, vv := range v {
							w.Header().Add(k, vv)
						}
					}
					w.WriteHeader(resp.StatusCode)
					io.Copy(w, resp.Body)
				}),
			}

			sshPort, _ := cmd.Flags().GetInt("ssh-port")
			sshServer := ssh.Server{
				Addr: fmt.Sprintf(":%d", sshPort),
				PublicKeyHandler: func(ctx ssh.Context, publicKey ssh.PublicKey) bool {
					key, err := keyText(publicKey)
					if err != nil {
						return false
					}
					ctx.SetValue("key", key)
					return true
				},
				RequestHandlers: map[string]ssh.RequestHandler{
					"user": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
						user, err := db.UserFromContext(ctx)
						if err != nil {
							slog.Info("no user found", slog.String("error", err.Error()))
							return false, nil
						}

						return true, gossh.Marshal(UserResponse{
							ID:    user.PublicID,
							Name:  user.Name,
							Email: user.Email,
						})

					},
					"signup": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
						// if the user already exists, it means they are already authenticated
						if _, err := db.UserFromContext(ctx); err == nil {
							return false, nil
						}

						var params SignupParams
						if err := gossh.Unmarshal(req.Payload, &params); err != nil {
							return false, nil
						}

						if err := isValidUsername(params.Username); err != nil {
							return false, nil
						}

						if _, err := mail.ParseAddress(params.Email); err != nil {
							return false, nil
						}

						if err := db.WrapTransaction(func(tx *sql.Tx) error {
							if db.selectUserWithEmail(tx, params.Email).Scan() != sql.ErrNoRows {
								return errors.New("user already exists")
							}

							if db.selectUserWithName(tx, params.Username).Scan() != sql.ErrNoRows {
								return errors.New("username already exists")
							}

							return nil
						}); err != nil {
							return false, nil
						}

						code, err := generateVerificationCode()
						if err != nil {
							return false, nil
						}

						if err := sendVerificationCode(valtownClient, params.Email, code); err != nil {
							return false, nil
						}

						conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
						codeOk, payload, err := conn.SendRequest("code", true, nil)
						if err != nil {
							return false, nil
						}
						if !codeOk {
							return false, nil
						}

						var verifyParams VerifyEmailParams
						if err := gossh.Unmarshal(payload, &verifyParams); err != nil {
							return false, nil
						}

						if verifyParams.Code != code {
							return false, nil
						}

						key, ok := ctx.Value("key").(string)
						if !ok {
							return false, nil
						}

						if err := db.WrapTransaction(func(tx *sql.Tx) error {
							return db.createUser(tx, key, params.Email, params.Username)
						}); err != nil {
							return false, nil
						}

						return true, nil
					},
					"login": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
						var params LoginParams
						if err := gossh.Unmarshal(req.Payload, &params); err != nil {
							return false, nil
						}

						var user *User
						if err := db.WrapTransaction(func(tx *sql.Tx) error {
							row := db.selectUserWithEmail(tx, params.Email)
							u, err := db.scanUser(row)
							if err != nil {
								return err
							}
							user = u

							return nil
						}); err != nil {
							return false, nil
						}

						code, err := generateVerificationCode()
						if err != nil {
							return false, nil
						}

						if err := sendVerificationCode(valtownClient, params.Email, code); err != nil {
							return false, nil
						}

						conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
						codeOk, payload, err := conn.SendRequest("code", true, nil)
						if err != nil {
							return false, nil
						}
						if !codeOk {
							return false, nil
						}

						var verifyParams VerifyEmailParams
						if err := gossh.Unmarshal(payload, &verifyParams); err != nil {
							return false, nil
						}

						if verifyParams.Code != code {
							return false, nil
						}

						key, ok := ctx.Value("key").(string)
						if !ok {
							return false, nil
						}

						if err := db.WrapTransaction(func(tx *sql.Tx) error {
							return db.insertPublicKey(tx, user.ID, key)
						}); err != nil {
							return false, nil
						}

						return true, nil
					},
					"logout": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
						user, err := db.UserFromContext(ctx)
						if err != nil {
							return false, nil
						}

						key := ctx.Value("key").(string)
						if err := db.WrapTransaction(func(tx *sql.Tx) error {
							return db.deletePublicKey(tx, user.ID, key)
						}); err != nil {
							return false, nil
						}

						return true, nil
					},
					"smallweb-forward": forwarder.HandleSSHRequest,
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

	cmd.Flags().Int("ssh-port", 2222, "port for the ssh server")
	cmd.Flags().Int("http-port", 8000, "port for the http server")

	return cmd
}

// Forwarder can be enabled by creating a Forwarder and
// adding the HandleSSHRequest callback to the server's RequestHandlers under
// tcpip-forward and cancel-tcpip-forward.
type Forwarder struct {
	db       *DB
	forwards map[string]int
	sync.Mutex
}

func (me *Forwarder) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	user, err := me.db.UserFromContext(ctx)
	if err != nil {
		slog.Info("no user found", slog.String("error", err.Error()))
		return false, nil
	}

	freeport, err := GetFreePort()
	if err != nil {
		return false, nil
	}

	addr := fmt.Sprintf("127.0.0.1:%d", freeport)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false, nil
	}

	me.Lock()
	me.forwards[user.Name] = freeport
	me.Unlock()
	go func() {
		<-ctx.Done()
		me.Lock()
		_, ok := me.forwards[user.Name]
		me.Unlock()
		if ok {
			ln.Close()
		}
	}()

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				// TODO: log accept failure
				break
			}

			go func() {
				conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
				ch, reqs, err := conn.OpenChannel("forwarded-smallweb", nil)
				if err != nil {
					// TODO: log failure to open channel
					log.Println(err)
					c.Close()
					return
				}
				go gossh.DiscardRequests(reqs)
				go func() {
					defer ch.Close()
					defer c.Close()
					io.Copy(ch, c)
				}()
				go func() {
					defer ch.Close()
					defer c.Close()
					io.Copy(c, ch)
				}()
			}()
		}
		me.Lock()
		delete(me.forwards, addr)
		me.Unlock()
	}()
	return true, nil
}

type RequestHeader struct {
	Method  string
	Url     string
	Headers [][]string
}

func (rh *RequestHeader) App() string {
	for _, v := range rh.Headers {
		if v[0] == "X-Smallweb-App" {
			return v[1]
		}
	}
	return ""
}

type ResponseHeader struct {
	Code    int
	Headers [][]string
}

// keyText is the base64 encoded public key for the glider.Session.
func keyText(publicKey gossh.PublicKey) (string, error) {
	kb := base64.StdEncoding.EncodeToString(publicKey.Marshal())
	return fmt.Sprintf("%s %s", publicKey.Type(), kb), nil
}

func isValidUsername(username string) error {
	// Check length
	if len(username) < 3 || len(username) > 15 {
		return fmt.Errorf("username must be between 3 and 15 characters")
	}

	// Check if it only contains alphanumeric characters
	alnumRegex := regexp.MustCompile(`^[a-z][a-z0-9]+$`)
	if !alnumRegex.MatchString(username) {
		return fmt.Errorf("username must only contain lowercase letters and numbers")
	}

	return nil
}

func generateVerificationCode() (string, error) {
	return generateRandomString(8, "0123456789")
}

func generateRandomString(length int, alphabet string) (string, error) {
	result := make([]byte, length)
	alphabetLength := big.NewInt(int64(len(alphabet)))

	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, alphabetLength)
		if err != nil {
			return "", err
		}
		result[i] = alphabet[index.Int64()]
	}

	return string(result), nil
}

type ValTownClient struct {
	token string
}

func NewValTownClient(token string) *ValTownClient {
	return &ValTownClient{
		token: token,
	}
}

func (me *ValTownClient) SendEmail(email Email) error {
	email.From = "pomdtr.smallweb@valtown.email"
	body, err := json.Marshal(email)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.val.town/v1/email", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", me.token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 202 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("could not send email: %v", string(msg))
	}

	return nil
}

func sendVerificationCode(client *ValTownClient, email string, code string) error {
	err := client.SendEmail(Email{
		To:      email,
		Subject: "Smallweb signup code",
		Text:    fmt.Sprintf("Your Smallweb signup code is: %s", code),
	})

	return err
}
