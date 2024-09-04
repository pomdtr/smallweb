package cmd

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	"github.com/adrg/xdg"
	"github.com/gobwas/glob"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/database"
	"github.com/pomdtr/smallweb/term"
	"github.com/pomdtr/smallweb/utils"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/webdav"
	"golang.org/x/oauth2"
)

type AuthMiddleware struct {
	db *sql.DB
}

func (me *AuthMiddleware) CreateSession(email string, domain string) (string, error) {
	sessionID, err := gonanoid.New()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := database.Session{
		ID:        sessionID,
		Email:     email,
		Domain:    domain,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	}

	if err := database.InsertSession(me.db, &session); err != nil {
		return "", fmt.Errorf("failed to insert session: %w", err)
	}

	return sessionID, nil
}

func (me *AuthMiddleware) DeleteSession(sessionID string) error {
	if err := database.DeleteSession(me.db, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (me *AuthMiddleware) GetSession(sessionID string, domain string) (database.Session, error) {
	session, err := database.GetSession(me.db, sessionID)
	if err != nil {
		return database.Session{}, fmt.Errorf("failed to get session: %w", err)
	}

	if session.Domain != domain {
		return database.Session{}, fmt.Errorf("session not found")
	}

	return *session, nil
}

func (me *AuthMiddleware) ExtendSession(sessionID string, expiresAt time.Time) error {
	session, err := database.GetSession(me.db, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.ExpiresAt = expiresAt
	if err := database.UpdateSession(me.db, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (me *AuthMiddleware) Wrap(next http.Handler, email string) http.Handler {
	sessionCookieName := "smallweb-session"
	oauthCookieName := "smallweb-oauth-store"
	type oauthStore struct {
		State    string `json:"state"`
		Redirect string `json:"redirect"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oauth2Config := oauth2.Config{
			ClientID: fmt.Sprintf("https://%s/", r.Host),
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://lastlogin.io/auth",
				TokenURL:  "https://lastlogin.io/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
			Scopes:      []string{"email"},
			RedirectURL: fmt.Sprintf("https://%s/_smallweb/auth/callback", r.Host),
		}

		username, _, ok := r.BasicAuth()
		if ok {
			tokens, err := database.ListTokens(me.db)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			for _, t := range tokens {
				if bcrypt.CompareHashAndPassword([]byte(t.Hash), []byte(username)) == nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		authorization := r.Header.Get("Authorization")
		if authorization != "" {
			if !strings.HasPrefix(authorization, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			}

			token := strings.Trim(strings.TrimPrefix(authorization, "Bearer "), " ")

			tokens, err := database.ListTokens(me.db)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			for _, t := range tokens {
				if bcrypt.CompareHashAndPassword([]byte(t.Hash), []byte(token)) == nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/_smallweb/auth/login" {
			query := r.URL.Query()
			state, err := generateToken(16)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			store := oauthStore{
				State:    state,
				Redirect: query.Get("redirect"),
			}

			value, err := json.Marshal(store)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

			http.SetCookie(w, &http.Cookie{
				Name:     oauthCookieName,
				Value:    url.QueryEscape(string(value)),
				Expires:  time.Now().Add(5 * time.Minute),
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			})

			url := oauth2Config.AuthCodeURL(state)
			http.Redirect(w, r, url, http.StatusSeeOther)
			return
		}

		if r.URL.Path == "/_smallweb/auth/callback" {
			query := r.URL.Query()
			oauthCookie, err := r.Cookie(oauthCookieName)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			var oauthStore oauthStore
			value, err := url.QueryUnescape(oauthCookie.Value)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if err := json.Unmarshal([]byte(value), &oauthStore); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if query.Get("state") != oauthStore.State {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			code := query.Get("code")
			if code == "" {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			token, err := oauth2Config.Exchange(r.Context(), code)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			req, err := http.NewRequest("GET", "https://lastlogin.io/userinfo", nil)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("userinfo request failed: %s", resp.Status)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			var userinfo struct {
				Email string `json:"email"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&userinfo); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			sessionID, err := me.CreateSession(userinfo.Email, r.Host)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// delete oauth cookie
			http.SetCookie(w, &http.Cookie{
				Name:     oauthCookieName,
				Expires:  time.Now().Add(-1 * time.Hour),
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			})

			// set session cookie
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    sessionID,
				Expires:  time.Now().Add(14 * 24 * time.Hour),
				HttpOnly: true,
				Secure:   true,
				Path:     "/",
			})

			http.Redirect(w, r, oauthStore.Redirect, http.StatusSeeOther)
			return
		}

		if r.URL.Path == "/_smallweb/auth/logout" {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if err := me.DeleteSession(cookie.Value); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Expires:  time.Now().Add(-1 * time.Hour),
				HttpOnly: true,
				Secure:   true,
				Path:     "/",
			})

			redirect := r.URL.Query().Get("redirect")
			if redirect == "" {
				redirect = fmt.Sprintf("https://%s/", r.Host)
			}

			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Redirect(w, r, fmt.Sprintf("/_smallweb/auth/login?redirect=%s", r.URL.Path), http.StatusSeeOther)
			return
		}

		session, err := me.GetSession(cookie.Value, r.Host)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Expires:  time.Now().Add(-1 * time.Hour),
				HttpOnly: true,
				Secure:   true,
			})

			http.Redirect(w, r, fmt.Sprintf("/_smallweb/auth/login?redirect=%s", r.URL.Path), http.StatusSeeOther)
			return
		}

		if time.Now().After(session.ExpiresAt) {
			if err := me.DeleteSession(cookie.Value); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Expires:  time.Now().Add(-1 * time.Hour),
				HttpOnly: true,
				Secure:   true,
			})

			http.Redirect(w, r, fmt.Sprintf("/_smallweb/auth/login?redirect=%s", r.URL.Path), http.StatusSeeOther)
			return
		}

		if session.Email != email {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// if session is near expiration, extend it
		if time.Now().Add(7 * 24 * time.Hour).After(session.ExpiresAt) {
			if err := me.ExtendSession(cookie.Value, time.Now().Add(14*24*time.Hour)); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			return
		}

		next.ServeHTTP(w, r)
	})
}

func NewCmdUp(db *sql.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			domain := k.String("domain")
			port := k.Int("port")
			cert := k.String("cert")
			key := k.String("key")

			if port == 0 {
				if cert != "" || key != "" {
					port = 443
				} else {
					port = 7777
				}
			}

			cliHandler, err := term.NewHandler(rootDir, k.String("editor"))
			if err != nil {
				return fmt.Errorf("failed to create cli handler: %w", err)
			}

			webdavHandler := &webdav.Handler{
				FileSystem: webdav.Dir(utils.ExpandTilde(k.String("dir"))),
				LockSystem: webdav.NewMemLS(),
			}

			sessionDBPath := filepath.Join(xdg.DataHome, "smallweb", "sessions.json")
			if err := os.MkdirAll(filepath.Dir(sessionDBPath), 0755); err != nil {
				return fmt.Errorf("failed to create session database directory: %w", err)
			}

			authMiddleware := AuthMiddleware{db}
			if err != nil {
				return fmt.Errorf("failed to create auth middleware: %w", err)
			}

			addr := fmt.Sprintf("%s:%d", k.String("host"), port)
			server := http.Server{
				Addr: addr,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Host == domain {
						target := r.URL
						target.Scheme = "https"
						target.Host = "www." + domain
						http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
					}

					if r.Host == fmt.Sprintf("webdav.%s", domain) {
						handler := authMiddleware.Wrap(webdavHandler, k.String("email"))
						handler.ServeHTTP(w, r)
						return
					}

					if r.Host == fmt.Sprintf("cli.%s", domain) {
						handler := authMiddleware.Wrap(cliHandler, k.String("email"))
						handler.ServeHTTP(w, r)
						return
					}

					var appDir string
					if strings.HasSuffix(r.Host, fmt.Sprintf(".%s", domain)) {
						appname := strings.TrimSuffix(r.Host, fmt.Sprintf(".%s", domain))
						appDir = filepath.Join(rootDir, appname)
						if !utils.FileExists(appDir) {
							w.WriteHeader(http.StatusNotFound)
							return
						}
					} else {
						for _, appname := range ListApps(rootDir) {
							cnamePath := filepath.Join(rootDir, appname, "CNAME")
							if !utils.FileExists("CNAME") {
								continue
							}

							cnameBytes, err := os.ReadFile(cnamePath)
							if err != nil {
								continue
							}

							if r.Host != string(cnameBytes) {
								continue
							}

							appDir = filepath.Join(rootDir, appname)
						}

						if appDir == "" {
							log.Printf("App not found for %s", r.Host)
							w.WriteHeader(http.StatusNotFound)
							return
						}
					}

					a, err := app.NewApp(appDir, r.Host, k.StringMap("env"))
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					if err := a.Start(); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					defer a.Stop()

					var handler http.Handler = a

					isPrivateRoute := a.Config.Private
					for _, publicRoute := range a.Config.PublicRoutes {
						glob := glob.MustCompile(publicRoute)
						if glob.Match(r.URL.Path) {
							isPrivateRoute = false
						}
					}

					for _, privateRoute := range a.Config.PrivateRoutes {
						glob := glob.MustCompile(privateRoute)
						if glob.Match(r.URL.Path) {
							isPrivateRoute = true
						}
					}

					if isPrivateRoute || strings.HasPrefix(r.URL.Path, "/_smallweb/auth") {
						handler = authMiddleware.Wrap(handler, k.String("email"))
					}

					handler.ServeHTTP(w, r)
				}),
			}

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			c.AddFunc("* * * * *", func() {
				rootDir := utils.ExpandTilde(k.String("dir"))
				rounded := time.Now().Truncate(time.Minute)
				apps := ListApps(rootDir)

				for _, appname := range apps {
					a, err := app.NewApp(appname, fmt.Sprintf("%s.%s", appname, domain), k.StringMap("env"))
					if err != nil {
						fmt.Println(err)
						continue
					}

					for _, job := range a.Config.Crons {
						sched, err := parser.Parse(job.Schedule)
						if err != nil {
							fmt.Println(err)
							continue
						}

						if sched.Next(rounded.Add(-1*time.Second)) != rounded {
							continue
						}

						go a.Run(job.Args)
					}

				}
			})

			go c.Start()

			if cert != "" || key != "" {
				if cert == "" {
					return fmt.Errorf("TLS certificate file is required")
				}

				if key == "" {
					return fmt.Errorf("TLS key file is required")
				}

				certificate, err := tls.LoadX509KeyPair(cert, key)
				if err != nil {
					return fmt.Errorf("failed to load TLS certificate and key: %w", err)
				}

				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{certificate},
					MinVersion:   tls.VersionTLS12,
				}

				server.TLSConfig = tlsConfig

				cmd.Printf("Serving %s from %s on %s\n", k.String("domain"), k.String("dir"), addr)
				return server.ListenAndServeTLS(cert, key)
			}

			cmd.Printf("Serving *.%s from %s on %s\n", k.String("domain"), k.String("dir"), addr)
			return server.ListenAndServe()
		},
	}

	return cmd
}
