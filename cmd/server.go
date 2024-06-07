package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/pomdtr/smallweb/storage"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

type contextKey struct {
	name string
}

var (
	ContextKeyEmail = &contextKey{"email"}
)

type LoginParams struct {
	Email string
}

type SignupParams struct {
	Email    string
	Username string
}

type SignupResponse struct {
	Username string
}

type LoginResponse struct {
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

type ErrorPayload struct {
	Message string
}

type ServerConfig struct {
	Host             string `env:"SMALLWEB_HOST" envDefault:"smallweb.run"`
	SSHPort          int    `env:"SMALLWEB_SSH_PORT" envDefault:"2222"`
	HttpPort         int    `env:"SMALLWEB_HTTP_PORT" envDefault:"8000"`
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

type AuthMiddleware struct {
	next http.Handler
	db   *storage.DB
}

func (me *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var oauthConfig = &oauth2.Config{
		ClientID:    fmt.Sprintf("https://%s/", r.Host),
		RedirectURL: fmt.Sprintf("https://%s/_smallweb/auth/callback", r.Host),
		Scopes:      []string{"email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://lastlogin.io/auth",
			TokenURL:  "https://lastlogin.io/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	if r.URL.Path == "/_smallweb/auth/callback" {
		state, err := r.Cookie("oauth_state")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}

		// delete the cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    "",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
			Expires:  time.Now(),
		})

		if r.FormValue("state") != state.Value {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		code := r.FormValue("code")
		token, err := oauthConfig.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		req, err := http.NewRequest(http.MethodGet, "https://lastlogin.io/userinfo", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var userinfo struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&userinfo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sessionID, err := me.db.CreateSession(userinfo.Email, r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "smallweb_session",
			Value:    sessionID,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
			MaxAge:   60 * 60 * 24 * 30,
		})

		http.Redirect(w, r, "https://"+r.Host, http.StatusTemporaryRedirect)
		return
	}

	if r.URL.Path == "/_smallweb/auth/logout" {
		http.SetCookie(w, &http.Cookie{
			Name:     "smallweb_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
			Expires:  time.Now(),
		})
	}

	sessionID, err := r.Cookie("smallweb_session")
	if err != nil {
		state := uuid.New().String()
		url := oauthConfig.AuthCodeURL(state)
		cookie := &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return
	}

	session, err := me.db.GetSession(sessionID.Value)
	if err != nil {
		state := uuid.New().String()
		url := oauthConfig.AuthCodeURL(state)
		cookie := &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return
	}

	if session.Host != r.Host {
		http.Error(w, "session not valid for this host", http.StatusUnauthorized)
		return
	}

	ctx := context.WithValue(r.Context(), ContextKeyEmail, session.Email)
	me.next.ServeHTTP(w, r.WithContext(ctx))
}

func NewAuthMiddleware(handler http.Handler, db *storage.DB) *AuthMiddleware {
	return &AuthMiddleware{
		next: handler,
		db:   db,
	}
}

type SubdomainHandler struct {
	db        *storage.DB
	forwarder *Forwarder
}

func (me *SubdomainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	visitorEmail := r.Context().Value(ContextKeyEmail).(string)
	subdomain := strings.Split(r.Host, ".")[0]
	parts := strings.Split(subdomain, "-")
	username := parts[len(parts)-1]
	user, err := me.db.GetUserWithName(username)

	if user.Email != visitorEmail {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	port, ok := me.forwarder.forwards[user.Name]
	if !ok {
		http.Error(w, fmt.Sprintf("User %s not found", user.Name), http.StatusNotFound)
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
	flusher := w.(http.Flusher)
	// Stream the response body to the client
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				http.Error(w, writeErr.Error(), http.StatusInternalServerError)
				return
			}
			flusher.Flush() // flush the buffer to the client
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}

func NewSubdomainHandler(db *storage.DB, forwarder *Forwarder) *SubdomainHandler {
	return &SubdomainHandler{
		db:        db,
		forwarder: forwarder,
	}
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

			db, err := storage.NewTursoDB(fmt.Sprintf("%s?authToken=%s", config.TursoDatabaseURL, config.TursoAuthToken))
			if err != nil {
				log.Fatalf("failed to open database: %v", err)
			}

			valtownClient := NewValTownClient(config.ValTownToken)

			forwarder := &Forwarder{
				db:       db,
				forwards: make(map[string]int),
			}

			subdomainHandler := NewSubdomainHandler(db, forwarder)

			httpServer := http.Server{
				Addr: fmt.Sprintf(":%d", config.HttpPort),
				Handler: NewAuthMiddleware(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						parts := strings.Split(r.Host, ".")
						if len(parts) == 3 {
							subdomainHandler.ServeHTTP(w, r)
							return
						}

						http.Error(w, "Not found", http.StatusNotFound)
					}), db,
				),
			}

			sshServer := ssh.Server{
				Addr: fmt.Sprintf(":%d", config.SSHPort),
				PublicKeyHandler: func(ctx ssh.Context, publicKey ssh.PublicKey) bool {
					log.Printf("attempted public key login: %s", publicKey.Type())
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
							return false, gossh.Marshal(ErrorPayload{Message: "no user found"})
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
							return false, gossh.Marshal(ErrorPayload{Message: "invalid payload"})
						}

						if err := isValidUsername(params.Username); err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "invalid username"})
						}

						if _, err := mail.ParseAddress(params.Email); err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "invalid email"})
						}

						if err := db.CheckUserInfo(params.Email, params.Username); err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "user already exists"})
						}

						code, err := generateVerificationCode()
						if err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "could not generate code"})
						}

						if err := sendVerificationCode(valtownClient, params.Email, code); err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "could not send code"})
						}

						conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
						codeOk, payload, err := conn.SendRequest("code", true, nil)
						if err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "could not send code"})
						}
						if !codeOk {
							return false, gossh.Marshal(ErrorPayload{Message: "could not send code"})
						}

						var verifyParams VerifyEmailParams
						if err := gossh.Unmarshal(payload, &verifyParams); err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "could not unmarshal code"})
						}

						if verifyParams.Code != code {
							return false, gossh.Marshal(ErrorPayload{Message: "invalid code"})
						}

						key, ok := ctx.Value("key").(string)
						if !ok {
							return false, gossh.Marshal(ErrorPayload{Message: "invalid key"})
						}

						if _, err := db.CreateUser(key, params.Email, params.Username); err != nil {
							return false, gossh.Marshal(ErrorPayload{Message: "could not create user"})
						}

						return true, nil
					},
					"login": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
						var params LoginParams
						if err := gossh.Unmarshal(req.Payload, &params); err != nil {
							return false, nil
						}

						user, err := db.GetUserWithEmail(params.Email)

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

						if err := db.AddUserPublicKey(user.ID, key); err != nil {
							return false, nil
						}

						return true, gossh.Marshal(LoginResponse{Username: user.Name})
					},
					"logout": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (ok bool, payload []byte) {
						user, err := db.UserFromContext(ctx)
						if err != nil {
							return false, nil
						}

						key := ctx.Value("key").(string)
						if err := db.DeleteUserPublicKey(user.ID, key); err != nil {
							return false, nil
						}

						return true, nil
					},
					"smallweb-forward": forwarder.HandleSSHRequest,
				},
			}

			slog.Info("starting ssh server", slog.Int("port", config.SSHPort))
			go sshServer.ListenAndServe()
			slog.Info("starting http server", slog.Int("port", config.HttpPort))
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

	return cmd
}

// Forwarder can be enabled by creating a Forwarder and
// adding the HandleSSHRequest callback to the server's RequestHandlers under
// tcpip-forward and cancel-tcpip-forward.
type Forwarder struct {
	db       *storage.DB
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

type Email struct {
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Cc      string `json:"cc,omitempty"`
	Bcc     string `json:"bcc,omitempty"`
	Subject string `json:"subject,omitempty"`
	Text    string `json:"text,omitempty"`
	Html    string `json:"html,omitempty"`
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
