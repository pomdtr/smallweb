package cmd

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	_ "embed"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/caddyserver/certmagic"
	"github.com/charmbracelet/ssh"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/charmbracelet/wish"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/creack/pty"
	"github.com/knadh/koanf/providers/file"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/mhale/smtpd"
	sloghttp "github.com/samber/slog-http"
	"go.uber.org/zap"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/sftp"
	"github.com/pomdtr/smallweb/internal/watcher"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"

	"github.com/pomdtr/smallweb/internal/utils"
	"github.com/pomdtr/smallweb/internal/worker"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	var flags struct {
		enableCrons   bool
		onDemandTLS   bool
		addr          string
		apiAddr       string
		sshAddr       string
		smtpAddr      string
		sshPrivateKey string
		tlsCert       string
		tlsKey        string
		logFormat     string
		logOutput     string
	}

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := checkDenoVersion(); err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var logger *slog.Logger

			var logOutput io.Writer
			switch flags.logOutput {
			case "stdout":
				logOutput = os.Stdout
			case "stderr", "":
				logOutput = os.Stderr
			default:
				logOutput = &lumberjack.Logger{
					Filename:   flags.logOutput,
					MaxSize:    10, // megabytes
					MaxBackups: 3,
				}
			}

			switch flags.logFormat {
			case "json":
				logger = slog.New(slog.NewJSONHandler(logOutput, &slog.HandlerOptions{}))
			case "text":
				logger = slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{}))
			case "pretty":
				logger = slog.New(tint.NewHandler(logOutput, &tint.Options{}))
			case "":
				if flags.logOutput == "stderr" && isatty.IsTerminal(os.Stderr.Fd()) || flags.logOutput == "stdout" && isatty.IsTerminal(os.Stdout.Fd()) {
					logger = slog.New(tint.NewHandler(logOutput, &tint.Options{}))
				} else {
					logger = slog.New(slog.NewJSONHandler(logOutput, &slog.HandlerOptions{}))
				}
			default:
				return fmt.Errorf("invalid log format: %s", flags.logFormat)
			}

			if k.String("dir") == "" {
				logger.Error("dir cannot be empty")
				return fmt.Errorf("dir cannot be empty")
			}

			if k.String("domain") == "" {
				return fmt.Errorf("domain cannot be empty")
			}

			handler := &Handler{
				workers: make(map[string]*worker.Worker),
				logger:  logger,
			}

			if issuer := k.String("oidc.issuer"); issuer != "" {
				issuerUrl, err := url.Parse(issuer)
				if err == nil {
					handler.oidcIssuerUrl = issuerUrl
				} else {
					logger.Error("failed to parse issuer url")
				}
			}

			watcher, err := watcher.NewWatcher(k.String("dir"), func() {
				fileProvider := file.Provider(utils.FindConfigPath(k.String("dir")))
				flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)

				k = koanf.New(".")
				_ = k.Load(fileProvider, utils.ConfigParser())
				_ = k.Load(envProvider, nil)
				_ = k.Load(flagProvider, nil)

				if issuer := k.String("oidc.issuer"); issuer != "" {
					issuerUrl, err := url.Parse(issuer)
					if err != nil {
						logger.Error("failed to parse issuer url")
						return
					}

					handler.oidcIssuerUrl = issuerUrl
					handler.oidcProvider = nil
				}
			})
			if err != nil {
				return fmt.Errorf("failed to create watcher: %w", err)
			}

			handler.watcher = watcher
			go watcher.Start()
			defer watcher.Stop()

			logMiddleware := sloghttp.NewWithConfig(logger.With("logger", "http"), sloghttp.Config{
				WithRequestID: false,
			})

			if flags.tlsCert != "" && flags.tlsKey != "" {
				cert, err := tls.LoadX509KeyPair(flags.tlsCert, flags.tlsKey)
				if err != nil {
					return fmt.Errorf("failed to load tls certificate: %w", err)
				}

				tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
				tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)

				addr := flags.addr
				if addr == "" {
					addr = ":443"
				}

				ln, err := getListener(addr, tlsConfig)
				if err != nil {
					return fmt.Errorf("failed to get listener: %w", err)
				}

				logger.Info("serving https", "domain", k.String("domain"), "dir", k.String("dir"), "addr", addr)
				go http.Serve(ln, logMiddleware(handler))
			} else if flags.onDemandTLS {
				certmagic.Default.Logger = zap.NewNop()
				certmagic.Default.OnDemand = &certmagic.OnDemandConfig{
					DecisionFunc: func(ctx context.Context, name string) error {
						if _, _, ok := lookupApp(name); ok {
							return nil
						}

						if _, err := os.Stat(filepath.Join(k.String("dir"), name)); err == nil {
							return nil
						}

						return fmt.Errorf("domain not found")
					},
				}
				logger.Info("serving on-demand https", "domain", k.String("domain"), "dir", k.String("dir"))
				go certmagic.HTTPS(nil, logMiddleware(handler))
			} else {
				addr := flags.addr
				if addr == "" {
					addr = ":7777"
				}

				ln, err := getListener(addr, nil)
				if err != nil {
					return fmt.Errorf("failed to get listener: %w", err)
				}

				logger.Info("serving http", "domain", k.String("domain"), "dir", k.String("dir"), "addr", addr)
				go http.Serve(ln, logMiddleware(handler))
			}

			if flags.enableCrons {
				logger.Info("starting cron jobs")
				crons := CronRunner(logger.With("logger", "cron"))
				crons.Start()
				defer crons.Stop()
			}

			if flags.smtpAddr != "" {
				handler := func(remoteAddr net.Addr, from string, to []string, data []byte) error {
					for _, recipient := range to {
						parts := strings.Split(recipient, "@")
						if len(parts) != 2 {
							logger.Error("invalid recipient", "recipient", recipient)
							continue
						}

						account, domain := parts[0], parts[1]
						if domain != k.String("domain") {
							logger.Error("invalid domain", "domain", domain)
							continue
						}

						a, err := app.LoadApp(account, k.String("dir"), k.String("domain"))
						if err != nil {
							logger.Error("failed to load app", "error", err)
							continue
						}

						worker := worker.NewWorker(a, k.Bool(fmt.Sprintf("apps.%s.admin", a.Name)), nil)
						if err := worker.SendEmail(context.Background(), data); err != nil {
							logger.Error("failed to send email", "error", err)
							continue
						}
					}

					return nil
				}

				logger.Info("starting smtp server", "addr", flags.smtpAddr)
				if flags.tlsCert != "" && flags.tlsKey != "" {
					go smtpd.ListenAndServeTLS(flags.smtpAddr, flags.tlsCert, flags.tlsKey, handler, "smallweb", k.String("domain"))
				} else {
					go smtpd.ListenAndServe(flags.smtpAddr, handler, "smallweb", k.String("domain"))
				}
			}

			if flags.sshAddr != "" {
				sshPrivateKeyPath := flags.sshPrivateKey
				if flags.sshPrivateKey == "" {
					homeDir, err := os.UserHomeDir()
					if err != nil {
						return fmt.Errorf("failed to get home directory: %w", err)
					}

					for _, keyType := range []string{"id_rsa", "id_ed25519"} {
						if _, err := os.Stat(filepath.Join(homeDir, ".ssh", keyType)); err == nil {
							sshPrivateKeyPath = filepath.Join(homeDir, ".ssh", keyType)
							break
						}
					}
				}

				if sshPrivateKeyPath == "" {
					return fmt.Errorf("ssh private key not found")
				}

				privateKeyBytes, err := os.ReadFile(sshPrivateKeyPath)
				if err != nil {
					return fmt.Errorf("failed to read private key: %w", err)
				}

				privateKey, err := gossh.ParseRawPrivateKey(privateKeyBytes)
				if err != nil {
					return fmt.Errorf("failed to parse private key: %w", err)
				}

				signer, err := gossh.NewSignerFromKey(privateKey)
				if err != nil {
					return fmt.Errorf("failed to create signer: %w", err)
				}

				authorizedKey := string(gossh.MarshalAuthorizedKey(signer.PublicKey()))
				sshLogger := logger.With("logger", "ssh")
				srv, err := wish.NewServer(
					wish.WithAddress(flags.sshAddr),
					wish.WithHostKeyPath(sshPrivateKeyPath),
					wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
						authorizedKeys := []string{authorizedKey}
						authorizedKeys = append(authorizedKeys, k.Strings("authorizedKeys")...)

						if ctx.User() != "_" {
							authorizedKeys = append(authorizedKeys, k.Strings(fmt.Sprintf("apps.%s.authorizedKeys", ctx.User()))...)
						}

						for _, authorizedKey := range authorizedKeys {
							k, _, _, _, err := gossh.ParseAuthorizedKey([]byte(authorizedKey))
							if err != nil {
								continue
							}

							if ssh.KeysEqual(k, key) {
								return true
							}
						}

						return false
					}),
					sftp.SSHOption(k.String("dir"), nil),
					wish.WithMiddleware(
						func(next ssh.Handler) ssh.Handler {
							return func(sess ssh.Session) {
								var cmd *exec.Cmd
								if sess.User() != "_" {
									a, err := app.LoadApp(sess.User(), k.String("dir"), k.String("domain"))
									if err != nil {
										fmt.Fprintf(sess, "failed to load app: %v\n", err)
										sess.Exit(1)
										return
									}

									wk := worker.NewWorker(a, k.Bool(fmt.Sprintf("apps.%s.admin", a.Name)), nil)
									c, err := wk.Command(sess.Context(), sess.Command())
									if err != nil {
										fmt.Fprintf(sess, "failed to get command: %v\n", err)
										sess.Exit(1)
										return
									}

									cmd = c
								} else {
									execPath, err := os.Executable()
									if err != nil {
										fmt.Fprintf(sess.Stderr(), "failed to get executable path: %v\n", err)
										sess.Exit(1)
										return
									}

									cmd = exec.Command(execPath, "--dir", k.String("dir"), "--domain", k.String("domain"))
									cmd.Args = append(cmd.Args, sess.Command()...)
									cmd.Env = os.Environ()
									cmd.Env = append(cmd.Env, "SMALLWEB_DISABLE_CUSTOM_COMMANDS=true")
								}

								ptyReq, winCh, isPty := sess.Pty()
								if isPty {
									cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term)
									f, err := pty.Start(cmd)
									if err != nil {
										fmt.Fprintf(sess, "failed to start command: %v\n", err)
										sess.Exit(1)
									}

									go func() {
										for win := range winCh {
											pty.Setsize(f, &pty.Winsize{
												Rows: uint16(win.Height),
												Cols: uint16(win.Width),
											})
										}
									}()

									go func() {
										io.Copy(sess, f)
									}()

									go func() {
										io.Copy(f, sess)
									}()

									if err := cmd.Wait(); err != nil {
										var exitErr *exec.ExitError
										if errors.As(err, &exitErr) {
											sess.Exit(exitErr.ExitCode())
											return
										}

										fmt.Fprintf(sess, "failed to run command: %v", err)
										sess.Exit(1)
										return
									}

									return
								}

								cmd.Stdout = sess
								cmd.Stderr = sess.Stderr()
								stdin, err := cmd.StdinPipe()
								if err != nil {
									fmt.Fprintf(sess, "failed to get stdin: %v\n", err)
									sess.Exit(1)
									return
								}

								go func() {
									defer stdin.Close()
									io.Copy(stdin, sess)
								}()

								if err := cmd.Run(); err != nil {
									var exitErr *exec.ExitError
									if errors.As(err, &exitErr) {
										sess.Exit(exitErr.ExitCode())
										return
									}

									fmt.Fprintf(sess, "failed to run command: %v", err)
									sess.Exit(1)
									return
								}
							}
						},
						func(next ssh.Handler) ssh.Handler {
							return func(sess ssh.Session) {
								sshLogger.Info(
									"ssh connection",
									"user", sess.User(),
									"remote addr", sess.RemoteAddr().String(),
									"command", sess.Command(),
								)
								next(sess)
							}
						},
					),
				)

				if err != nil {
					logger.Error("failed to create ssh server", "error", err)
					return fmt.Errorf("failed to create ssh server: %w", err)
				}

				logger.Info("serving ssh", "addr", flags.sshAddr)
				go srv.ListenAndServe()
			}

			// sigint handling
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.addr, "addr", "", "address to listen on")
	cmd.Flags().StringVar(&flags.sshAddr, "ssh-addr", "", "address to listen on for ssh/sftp")
	cmd.Flags().StringVar(&flags.smtpAddr, "smtp-addr", "", "address to listen on for smtp")
	cmd.Flags().StringVar(&flags.sshPrivateKey, "ssh-private-key", "", "ssh private key")
	cmd.Flags().StringVar(&flags.sshPrivateKey, "ssh-host-key", "", "ssh host key")
	cmd.Flags().StringVar(&flags.tlsCert, "tls-cert", "", "tls certificate file")
	cmd.Flags().StringVar(&flags.tlsKey, "tls-key", "", "tls key file")
	cmd.Flags().BoolVar(&flags.onDemandTLS, "on-demand-tls", false, "enable on-demand tls")
	cmd.Flags().StringVar(&flags.logFormat, "log-format", "", "log format (json, text or pretty)")
	cmd.Flags().StringVar(&flags.logOutput, "log-output", "stderr", "log output (stdout, stderr or filepath)")
	cmd.Flags().BoolVar(&flags.enableCrons, "enable-crons", false, "enable cron jobs")

	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "tls-cert")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "tls-key")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "addr")

	cmd.MarkFlagsRequiredTogether("tls-cert", "tls-key")

	return cmd
}

func getListener(addr string, config *tls.Config) (net.Listener, error) {
	if strings.HasPrefix(addr, "unix/") {
		socketPath := strings.TrimPrefix(addr, "unix/")

		if utils.FileExists(socketPath) {
			if err := os.Remove(socketPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing socket: %v", err)
			}
		}

		if config != nil {
			return tls.Listen("unix", socketPath, config)
		}

		return net.Listen("unix", socketPath)
	}

	addr = strings.TrimPrefix(addr, "tcp/")
	if config != nil {
		return tls.Listen("tcp", addr, config)
	}

	return net.Listen("tcp", addr)
}

type Handler struct {
	watcher       *watcher.Watcher
	logger        *slog.Logger
	workerMu      sync.Mutex
	workers       map[string]*worker.Worker
	oidcMu        sync.Mutex
	oidcIssuerUrl *url.URL
	oidcProvider  *oidc.Provider
}

type AuthData struct {
	State        string `json:"state"`
	SuccessURL   string `json:"success_url"`
	CodeVerifier string `json:"code_verifier"`
}

type IssuerConfig struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JwksUri               string `json:"jwks_uri"`
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hostname, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		hostname = r.Host
	}

	appname, redirect, ok := lookupApp(hostname)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("No app found for hostname %s", hostname)))
		return
	}

	if redirect {
		target := r.URL
		target.Scheme = r.Header.Get("X-Forwarded-Proto")
		if target.Scheme == "" {
			target.Scheme = "http"
		}

		target.Host = fmt.Sprintf("%s.%s", appname, r.Host)
		http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
		return
	}

	if me.oidcIssuerUrl != nil && me.oidcIssuerUrl.Host == r.Host {
		wk, err := me.GetWorker(appname, k.String("dir"), k.String("domain"))
		if err != nil {
			if errors.Is(err, app.ErrAppNotFound) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(fmt.Sprintf("No app found for host %s", r.Host)))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to get worker: %v", err)
			return
		}

		wk.ServeHTTP(w, r)
		return
	}

	if r.URL.Path == "/_smallweb/signin" {
		if me.oidcIssuerUrl == nil {
			http.Error(w, "oidc issuer url not set", http.StatusInternalServerError)
			return
		}

		oauth2Config, err := me.Oauth2Config(r.Host)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get oauth2 config: %v", err), http.StatusInternalServerError)
			return
		}

		var successURL string
		if param := r.URL.Query().Get("success_url"); param != "" {
			successURL = fmt.Sprintf("https://%s%s", r.Host, param)
		} else if r.Header.Get("Referer") != "" {
			successURL = r.Header.Get("Referer")
		} else {
			successURL = fmt.Sprintf("https://%s/", r.Host)
		}

		state := rand.Text()
		verifier := oauth2.GenerateVerifier()
		authData := AuthData{
			State:        state,
			SuccessURL:   successURL,
			CodeVerifier: verifier,
		}

		// Marshal the struct to JSON
		jsonData, err := json.Marshal(authData)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		encodedData := base64.StdEncoding.EncodeToString(jsonData)
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_data",
			Value:    encodedData,
			Secure:   true,
			HttpOnly: true,
			MaxAge:   5 * 60,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, oauth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusTemporaryRedirect)
		return
	}

	if r.URL.Path == "/_smallweb/signout" {
		if me.oidcIssuerUrl == nil {
			http.Error(w, "oidc issuer url not set", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "id_token",
			Secure:   true,
			HttpOnly: true,
			Path:     "/",
			MaxAge:   -1,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Secure:   true,
			HttpOnly: true,
			Path:     "/",
			MaxAge:   -1,
		})

		var successUrl string
		if param := r.URL.Query().Get("success_url"); param != "" {
			successUrl = fmt.Sprintf("https://%s%s", r.Host, param)
		} else if r.Header.Get("Referer") != "" {
			successUrl = r.Header.Get("Referer")
		} else {
			successUrl = fmt.Sprintf("https://%s/", r.Host)
		}

		http.Redirect(w, r, successUrl, http.StatusTemporaryRedirect)
		return
	}

	if r.URL.Path == "/_smallweb/oauth/callback" {
		if me.oidcIssuerUrl == nil {
			http.Error(w, "oidc issuer url not set", http.StatusInternalServerError)
			return
		}

		oauth2Config, err := me.Oauth2Config(r.Host)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get oauth2 config: %v", err), http.StatusInternalServerError)
			return
		}

		authCookie, err := r.Cookie("oauth_data")
		if err != nil {
			http.Error(w, "state cookie not found", http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_data",
			Secure:   true,
			HttpOnly: true,
			Path:     "/",
			MaxAge:   -1,
		})

		decodedData, err := base64.StdEncoding.DecodeString(authCookie.Value)
		if err != nil {
			http.Error(w, "failed to decode state cookie", http.StatusUnauthorized)
			return
		}

		var authData AuthData
		if err := json.Unmarshal(decodedData, &authData); err != nil {
			http.Error(w, "failed to unmarshal state cookie", http.StatusUnauthorized)
			return
		}

		if authData.State != r.URL.Query().Get("state") {
			http.Error(w, "invalid state", http.StatusUnauthorized)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "oauth code not found", http.StatusUnauthorized)
			return
		}

		oauth2Token, err := oauth2Config.Exchange(r.Context(), code, oauth2.VerifierOption(authData.CodeVerifier))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to exchange code: %v", err), http.StatusInternalServerError)
			return
		}

		idToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "id token not found", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "id_token",
			Value:    idToken,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
			HttpOnly: true,
			Path:     "/",
			MaxAge:   34560000,
		})

		if oauth2Token.RefreshToken != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    oauth2Token.RefreshToken,
				SameSite: http.SameSiteLaxMode,
				Secure:   true,
				HttpOnly: true,
				Path:     "/",
				MaxAge:   34560000,
			})
		}

		http.Redirect(w, r, authData.SuccessURL, http.StatusTemporaryRedirect)
		return
	}

	wk, err := me.GetWorker(appname, k.String("dir"), k.String("domain"))
	if err != nil {
		if errors.Is(err, app.ErrAppNotFound) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("No app found for host %s", r.Host)))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to get worker: %v", err)
		return
	}

	claims, err := me.extractClaims(r)
	if err != nil && isRoutePrivate(wk.App, r.URL.Path) {
		if me.oidcIssuerUrl == nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if !errors.Is(err, &oidc.TokenExpiredError{}) {
			http.Redirect(w, r, fmt.Sprintf("https://%s/_smallweb/signin?success_url=%s", r.Host, r.URL.Path), http.StatusTemporaryRedirect)
			return
		}

		var expiredErr *oidc.TokenExpiredError
		if errors.As(err, &expiredErr) {
			refreshTokenCookie, err := r.Cookie("refresh_token")
			if err != nil {
				http.Redirect(w, r, fmt.Sprintf("https://%s/_smallweb/signin?success_url=%s", r.Host, r.URL.Path), http.StatusTemporaryRedirect)
				return
			}

			oauth2Config, err := me.Oauth2Config(r.Host)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to get oauth2 config: %v", err), http.StatusInternalServerError)
				return
			}

			tokenSource := oauth2Config.TokenSource(context.Background(), &oauth2.Token{RefreshToken: refreshTokenCookie.Value})
			oauth2Token, err := tokenSource.Token()
			if err != nil {
				http.Redirect(w, r, fmt.Sprintf("https://%s/_smallweb/signin?success_url=%s", r.Host, r.URL.Path), http.StatusTemporaryRedirect)
				return
			}

			rawIdToken, ok := oauth2Token.Extra("id_token").(string)
			if !ok {
				http.Redirect(w, r, fmt.Sprintf("https://%s/_smallweb/signin?success_url=%s", r.Host, r.URL.Path), http.StatusTemporaryRedirect)
				return
			}

			provider, ok := me.Provider()
			if !ok {
				http.Error(w, "oidc provider not found", http.StatusInternalServerError)
				return
			}

			verifier := provider.Verifier(&oidc.Config{ClientID: r.Host})
			idToken, err := verifier.Verify(r.Context(), rawIdToken)
			if err != nil {
				http.Redirect(w, r, fmt.Sprintf("https://%s/_smallweb/signin?success_url=%s", r.Host, r.URL.Path), http.StatusTemporaryRedirect)
				return
			}

			if err := idToken.Claims(&claims); err != nil {
				http.Redirect(w, r, fmt.Sprintf("https://%s/_smallweb/signin?success_url=%s", r.Host, r.URL.Path), http.StatusTemporaryRedirect)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "id_token",
				Value:    rawIdToken,
				SameSite: http.SameSiteLaxMode,
				Secure:   true,
				HttpOnly: true,
				Path:     "/",
				MaxAge:   34560000,
			})

			if oauth2Token.RefreshToken != "" {
				http.SetCookie(w, &http.Cookie{
					Name:     "refresh_token",
					Value:    oauth2Token.RefreshToken,
					SameSite: http.SameSiteLaxMode,
					Secure:   true,
					HttpOnly: true,
					Path:     "/",
					MaxAge:   34560000,
				})
			}
		}
	}

	if isRoutePrivate(wk.App, r.URL.Path) && !isAuthorized(appname, claims.Email, claims.Group) {
		if claims.Email == "" {
			http.Redirect(w, r, fmt.Sprintf("https://%s/_smallweb/signin?success_url=%s", r.Host, r.URL.Path), http.StatusTemporaryRedirect)
			return
		}

		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	r.Header.Set("Remote-User", claims.User)
	r.Header.Set("Remote-Email", claims.Email)
	r.Header.Set("Remote-Group", claims.Group)
	r.Header.Set("Remote-Name", claims.Name)

	wk.ServeHTTP(w, r)
}

func isRoutePrivate(a app.App, route string) bool {
	isPrivate := a.Config.Private

	for _, publicRoute := range a.Config.PublicRoutes {
		if ok, _ := doublestar.Match(publicRoute, route); ok {
			isPrivate = false
		}
	}

	for _, privateRoute := range a.Config.PrivateRoutes {
		if ok, _ := doublestar.Match(privateRoute, route); ok {
			isPrivate = true
		}
	}

	return isPrivate
}

func isAuthorized(appname string, email string, group string) bool {
	var authorizedEmails []string
	authorizedEmails = append(authorizedEmails, k.Strings("authorizedEmails")...)
	authorizedEmails = append(authorizedEmails, k.Strings(fmt.Sprintf("apps.%s.authorizedEmails", appname))...)

	for _, authorizedEmail := range authorizedEmails {
		if match, _ := doublestar.Match(authorizedEmail, email); match {
			return true
		}
	}

	var authorizedGroups []string
	authorizedGroups = append(authorizedGroups, k.Strings("authorizedGroups")...)
	authorizedGroups = append(authorizedGroups, k.Strings(fmt.Sprintf("apps.%s.authorizedGroups", appname))...)

	for _, authorizedGroup := range authorizedGroups {
		if match, _ := doublestar.Match(authorizedGroup, group); match {
			return true
		}
	}

	return false
}

type Claims struct {
	Email string
	Group string
	User  string
	Name  string
}

func (me *Handler) extractClaims(r *http.Request) (Claims, error) {
	provider, ok := me.Provider()
	if !ok {
		return Claims{
			Email: r.Header.Get("Remote-Email"),
			Group: r.Header.Get("Remote-Group"),
			User:  r.Header.Get("Remote-User"),
			Name:  r.Header.Get("Remote-Name"),
		}, nil
	}

	var rawIdToken string
	if auth := r.Header.Get("Authorization"); auth != "" {
		parts := strings.Split(strings.TrimSpace(auth), " ")
		if len(parts) != 2 {
			return Claims{}, fmt.Errorf("invalid authorization header")
		}

		if parts[0] != "Bearer" {
			return Claims{}, fmt.Errorf("invalid authorization header")
		}

		rawIdToken = parts[1]

	} else {
		idTokenCookie, err := r.Cookie("id_token")
		if err != nil {
			return Claims{}, fmt.Errorf("id token not found")
		}

		rawIdToken = idTokenCookie.Value
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: fmt.Sprintf("https://%s", r.Host)})
	idToken, err := verifier.Verify(r.Context(), rawIdToken)
	if err != nil {
		return Claims{}, fmt.Errorf("failed to verify id token: %v", err)
	}

	var userinfo Claims
	if err := idToken.Claims(&userinfo); err != nil {
		return Claims{}, fmt.Errorf("failed to extract claims: %v", err)
	}

	return userinfo, nil
}

func lookupApp(domain string) (app string, redirect bool, found bool) {
	for _, app := range k.MapKeys("apps") {
		if slices.Contains(k.Strings(fmt.Sprintf("apps.%s.additionalDomains", app)), domain) {
			return app, false, true
		}
	}

	if domain == k.String("domain") {
		return "www", true, true
	}

	if strings.HasSuffix(domain, fmt.Sprintf(".%s", k.String("domain"))) {
		return strings.TrimSuffix(domain, fmt.Sprintf(".%s", k.String("domain"))), false, true
	}

	for _, additionalDomain := range k.Strings("additionalDomains") {
		if domain == additionalDomain {
			return "www", true, true
		}

		if strings.HasSuffix(domain, fmt.Sprintf(".%s", additionalDomain)) {
			return strings.TrimSuffix(domain, fmt.Sprintf(".%s", additionalDomain)), false, true
		}
	}

	return "", false, false
}

func (me *Handler) Provider() (*oidc.Provider, bool) {
	me.oidcMu.Lock()
	defer me.oidcMu.Unlock()

	if me.oidcIssuerUrl == nil {
		return nil, false
	}

	if me.oidcProvider == nil {
		provider, err := oidc.NewProvider(context.Background(), me.oidcIssuerUrl.String())
		if err != nil {
			me.logger.Error("failed to create oidc provider", "error", err)
			return nil, false
		}

		me.oidcProvider = provider
	}

	return me.oidcProvider, true
}

func (me *Handler) Oauth2Config(host string) (*oauth2.Config, error) {
	clientID := fmt.Sprintf("https://%s", host)
	provider, ok := me.Provider()
	if !ok {
		return nil, fmt.Errorf("oidc provider not set")
	}

	return &oauth2.Config{
		ClientID:    clientID,
		Scopes:      []string{"openid", "email", "profile", "groups"},
		RedirectURL: fmt.Sprintf("https://%s/_smallweb/oauth/callback", host),
		Endpoint:    provider.Endpoint(),
	}, nil
}

func (me *Handler) GetWorker(appname string, rootDir, domain string) (*worker.Worker, error) {
	if wk, ok := me.workers[appname]; ok && wk.IsRunning() && me.watcher.GetAppMtime(appname).Before(wk.StartedAt) {
		return wk, nil
	}

	me.workerMu.Lock()
	defer me.workerMu.Unlock()

	a, err := app.LoadApp(appname, k.String("dir"), k.String("domain"))
	if err != nil {
		return nil, fmt.Errorf("failed to load app: %w", err)
	}

	wk := worker.NewWorker(a, k.Bool(fmt.Sprintf("apps.%s.admin", a.Name)), me.logger.With("logger", "console", "app", appname))
	if err := wk.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	me.workers[a.Name] = wk
	return wk, nil
}
