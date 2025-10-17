package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	_ "embed"

	gossh "golang.org/x/crypto/ssh"

	"github.com/caddyserver/certmagic"
	"github.com/charmbracelet/ssh"
	"github.com/creack/pty"
	"github.com/mhale/smtpd"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/charmbracelet/wish"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	sloghttp "github.com/samber/slog-http"
	"go.uber.org/zap"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/sftp"
	"github.com/pomdtr/smallweb/internal/watcher"

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
			default:
				if flags.logOutput == "stderr" && isatty.IsTerminal(os.Stderr.Fd()) || flags.logOutput == "stdout" && isatty.IsTerminal(os.Stdout.Fd()) {
					logger = slog.New(tint.NewHandler(logOutput, &tint.Options{}))
				} else {
					logger = slog.New(slog.NewJSONHandler(logOutput, &slog.HandlerOptions{}))
				}
			}

			sysLogger := logger.With("logger", "system")

			if k.String("dir") == "" {
				sysLogger.Error("dir cannot be empty")
				return ExitError{1}
			}

			handler := &Handler{
				rootDir: k.String("dir"),
				workers: make(map[string]*worker.Worker),
				logger:  logger,
			}

			watcher, err := watcher.NewWatcher(k.String("dir"))
			if err != nil {
				logger.Error("failed to create watcher", "err", err)
				return ExitError{1}
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
					sysLogger.Error("failed to load tls certificate", "error", err)
					return ExitError{1}
				}

				tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
				tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)

				addr := flags.addr
				if addr == "" {
					addr = ":443"
				}

				ln, err := getListener(addr, tlsConfig)
				if err != nil {
					sysLogger.Error("failed to get listener", "error", err)
					return ExitError{1}
				}

				logger.Info("serving https", "dir", k.String("dir"), "addr", addr)
				go http.Serve(ln, logMiddleware(handler))
			} else if flags.onDemandTLS {
				certmagic.Default.Logger = zap.NewNop()
				certmagic.Default.OnDemand = &certmagic.OnDemandConfig{
					DecisionFunc: func(ctx context.Context, name string) error {
						if _, _, ok := ResolveHostname(k.String("dir"), name); ok {
							return nil
						}

						return fmt.Errorf("domain not found")
					},
				}
				logger.Info("serving on-demand https", "dir", k.String("dir"))
				go certmagic.HTTPS(nil, logMiddleware(handler))
			} else {
				addr := flags.addr
				if addr == "" {
					addr = ":7777"
				}

				ln, err := getListener(addr, nil)
				if err != nil {
					sysLogger.Error("failed to get listener", "error", err)
					return ExitError{1}
				}

				logger.Info("serving http", "dir", k.String("dir"), "addr", addr)
				go http.Serve(ln, logMiddleware(handler))
			}

			if flags.enableCrons {
				logger.Info("starting cron jobs")
				crons := CronRunner(k.String("dir"), logger.With("logger", "cron"))
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

						a, err := app.LoadApp(k.String("dir"), parts[1], parts[0])
						if err != nil {
							logger.Error("failed to load app for recipient", "recipient", recipient, "error", err)
							continue
						}

						worker := worker.NewWorker(a, nil)
						if err := worker.SendEmail(context.Background(), data); err != nil {
							logger.Error("failed to send email", "error", err)
							continue
						}
					}

					return nil
				}

				logger.Info("starting smtp server", "addr", flags.smtpAddr)
				if flags.tlsCert != "" && flags.tlsKey != "" {
					go smtpd.ListenAndServeTLS(flags.smtpAddr, flags.tlsCert, flags.tlsKey, handler, "", "")
				} else {
					go smtpd.ListenAndServe(flags.smtpAddr, handler, "", "")
				}
			}

			if flags.sshAddr != "" {
				sshPrivateKeyPath := flags.sshPrivateKey
				if flags.sshPrivateKey == "" {
					homeDir, err := os.UserHomeDir()
					if err != nil {
						sysLogger.Error("failed to get home directory", "error", err)
						return ExitError{1}
					}

					for _, keyType := range []string{"id_rsa", "id_ed25519"} {
						if _, err := os.Stat(filepath.Join(homeDir, ".ssh", keyType)); err == nil {
							sshPrivateKeyPath = filepath.Join(homeDir, ".ssh", keyType)
							break
						}
					}
				}

				if sshPrivateKeyPath == "" {
					sysLogger.Error("ssh private key not found")
					return ExitError{1}
				}

				privateKeyBytes, err := os.ReadFile(sshPrivateKeyPath)
				if err != nil {
					sysLogger.Error("failed to read private key", "error", err)
					return ExitError{1}
				}

				privateKey, err := gossh.ParseRawPrivateKey(privateKeyBytes)
				if err != nil {
					sysLogger.Error("failed to parse private key", "error", err)
					return ExitError{1}
				}

				signer, err := gossh.NewSignerFromKey(privateKey)
				if err != nil {
					sysLogger.Error("failed to create signer", "error", err)
					return ExitError{1}
				}

				authorizedKey := string(gossh.MarshalAuthorizedKey(signer.PublicKey()))
				sshLogger := logger.With("logger", "ssh")
				srv, err := wish.NewServer(
					wish.WithAddress(flags.sshAddr),
					wish.WithHostKeyPath(sshPrivateKeyPath),
					wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
						authorizedKeys := []string{authorizedKey}
						homedir, err := os.UserHomeDir()
						if err != nil {
							sshLogger.Error("failed to get home directory", "error", err)
							return false
						}

						authorizedKeysPaths := []string{
							filepath.Join(homedir, ".ssh", "authorizedKeys"),
							filepath.Join(k.String("dir"), ".ssh", "authorized_keys"),
							filepath.Join(k.String("dir"), ctx.User(), ".ssh", "authorized_keys"),
						}

						for _, authorizedKeysPath := range authorizedKeysPaths {
							authorizedKeyBytes, err := os.ReadFile(authorizedKeysPath)
							if err != nil {
								continue
							}

							for len(authorizedKeyBytes) > 0 {
								pubKey, _, _, rest, err := gossh.ParseAuthorizedKey(authorizedKeyBytes)
								if err != nil {
									break
								}
								authorizedKeys = append(authorizedKeys, string(gossh.MarshalAuthorizedKey(pubKey)))
								authorizedKeyBytes = rest
							}
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
								execPath, err := os.Executable()
								if err != nil {
									fmt.Fprintf(sess, "failed to get executable path: %v\n", err)
								}

								cmd := exec.Command(execPath, "ssh", "--dir", k.String("dir"), "--domain", sess.User())
								cmd.Args = append(cmd.Args, sess.Command()...)
								cmd.Env = os.Environ()

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
					sysLogger.Error("failed to create ssh server", "error", err)
					return ExitError{1}
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

	cmd.Flags().String("dir", "", "directory containing the apps")
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
	rootDir  string
	watcher  *watcher.Watcher
	logger   *slog.Logger
	workerMu sync.Mutex
	workers  map[string]*worker.Worker
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

	domain, appname, ok := ResolveHostname(me.rootDir, hostname)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("No app found for host %s", r.Host)))
		return
	}

	wk, err := me.GetWorker(domain, appname)
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
}

func (me *Handler) GetWorker(domain string, appname string) (*worker.Worker, error) {
	if wk, ok := me.workers[domain]; ok && wk.IsRunning() && me.watcher.GetAppMtime(domain).Before(wk.StartedAt) {
		return wk, nil
	}

	me.workerMu.Lock()
	defer me.workerMu.Unlock()

	a, err := app.LoadApp(me.rootDir, domain, appname)
	if err != nil {
		return nil, fmt.Errorf("failed to load app: %w", err)
	}

	wk := worker.NewWorker(a, me.logger.With("logger", "console", "dir", domain))
	if err := wk.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	me.workers[a.Domain] = wk
	return wk, nil
}

func ResolveHostname(dir string, hostname string) (string, string, bool) {
	if _, err := os.Stat(filepath.Join(dir, hostname, "_")); err == nil {
		return "_", hostname, true
	}

	parts := strings.SplitN(hostname, ".", 2)
	if _, err := os.Stat(filepath.Join(dir, parts[1], parts[0])); err == nil {
		return parts[0], parts[1], true
	}

	return "", "", false
}

func ExtractScheme(r *http.Request) string {
	if scheme := r.URL.Query().Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}
