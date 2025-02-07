package cmd

import (
	"crypto/tls"
	"encoding/json"
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
	"slices"
	"strings"
	"sync"

	_ "embed"

	"github.com/adrg/xdg"
	"github.com/caddyserver/certmagic"
	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/libdns/acmedns"
	"github.com/picosh/pobj"
	"github.com/pomdtr/smallweb/storage"

	"github.com/picosh/send/protocols/rsync"
	"github.com/picosh/send/protocols/scp"
	"github.com/picosh/send/protocols/sftp"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/watcher"
	gossh "golang.org/x/crypto/ssh"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	var flags struct {
		cron        bool
		httpAddr    string
		sshAddr     string
		sshHostKey  string
		tlsCert     string
		tlsKey      string
		acmdnsCreds string
	}

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if k.String("domain") == "" {
				return fmt.Errorf("domain cannot be empty")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if k.String("domain") == "" {
				return fmt.Errorf("domain cannot be empty")
			}

			logFilename := utils.GetLogFilename(k.String("domain"))
			if err := os.MkdirAll(filepath.Dir(logFilename), 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %v", err)
			}

			httpLogger := utils.NewLogger(&lumberjack.Logger{
				Filename:   logFilename,
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     28,
			})

			watcher, err := watcher.NewWatcher(k.String("dir"))
			if err != nil {
				return fmt.Errorf("failed to create watcher: %v", err)
			}

			go watcher.Start()
			defer watcher.Stop()

			consoleLogger := slog.New(slog.NewJSONHandler(&lumberjack.Logger{
				Filename:   logFilename,
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     28,
			}, nil))

			handler := httpLogger.Middleware(&Handler{
				logger:  consoleLogger,
				watcher: watcher,
				workers: make(map[string]*worker.Worker),
			})

			if flags.acmdnsCreds != "" {
				credentialPath := filepath.Join(xdg.DataHome, "smallweb", "acmedns", "credentials.json")
				if _, err := os.Stat(credentialPath); err != nil {
					return fmt.Errorf("acme-dns file not found: %s", credentialPath)
				}

				configBytes, err := os.ReadFile(credentialPath)
				if err != nil {
					return fmt.Errorf("failed to read acme-dns file: %v", err)
				}

				var configs map[string]acmedns.DomainConfig
				if err := json.Unmarshal(configBytes, &configs); err != nil {
					return fmt.Errorf("failed to unmarshal acme-dns file: %v", err)
				}

				certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
					DNSManager: certmagic.DNSManager{
						DNSProvider: &acmedns.Provider{
							Configs: configs,
						},
					},
				}

				if k.String("email") != "" {
					certmagic.DefaultACME.Email = k.String("email")
				} else {
					certmagic.DefaultACME.Email = fmt.Sprintf("smallweb@%s", k.String("domain"))
				}

				domains := []string{k.String("domain"), fmt.Sprintf("*.%s", k.String("domain"))}
				for customDomain, target := range k.StringMap("customDomains") {
					if target == "*" {
						domains = append(domains, customDomain, fmt.Sprintf("*.%s", customDomain))
						continue
					}

					domains = append(domains, customDomain)
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Serving *.%s from %s...\n", k.String("domain"), utils.AddTilde(k.String("dir")))
				go certmagic.HTTPS([]string{
					k.String("domain"),
					fmt.Sprintf("*.%s", k.String("domain")),
				}, handler)
			} else {
				if email := k.String("email"); email != "" {
					certmagic.DefaultACME.Email = email
				}

				addr := flags.httpAddr
				if addr == "" {
					if flags.tlsCert != "" || flags.tlsKey != "" {
						addr = "0.0.0.0:443"
					} else {
						addr = "localhost:7777"
					}
				}

				listener, err := getListener(addr, flags.tlsCert, flags.tlsKey)
				if err != nil {
					return fmt.Errorf("failed to get listener: %v", err)
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Serving *.%s from %s on %s...\n", k.String("domain"), utils.AddTilde(k.String("dir")), addr)
				go http.Serve(listener, handler)
			}

			if flags.cron {
				fmt.Fprintln(cmd.ErrOrStderr(), "Starting cron jobs...")
				crons := CronRunner(cmd.OutOrStdout(), cmd.ErrOrStderr())
				crons.Start()
				defer crons.Stop()
			}

			if flags.sshAddr != "" {
				if !utils.FileExists(flags.sshHostKey) {
					_, err := keygen.New(flags.sshHostKey, keygen.WithWrite())
					if err != nil {
						return fmt.Errorf("failed to generate host key: %v", err)
					}
				}

				st := storage.StorageFS{
					Dir:    k.String("dir"),
					Logger: consoleLogger,
				}

				cfg := pobj.Config{
					Storage: &st,
					Logger:  consoleLogger,
				}

				handler := pobj.NewUploadAssetHandler(&cfg)

				srv, err := wish.NewServer(
					wish.WithAddress(flags.sshAddr),
					wish.WithHostKeyPath(flags.sshHostKey),
					wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
						authorizedKeyPaths := []string{filepath.Join(k.String("dir"), ".smallweb", "authorized_keys")}

						if ctx.User() != "_" {
							authorizedKeyPaths = append(authorizedKeyPaths, filepath.Join(k.String("dir"), ctx.User(), "authorized_keys"))
						}

						for _, authorizedKeysPath := range authorizedKeyPaths {
							if _, err := os.Stat(authorizedKeysPath); err != nil {
								return false
							}

							authorizedKeysBytes, err := os.ReadFile(authorizedKeysPath)
							if err != nil {
								return false
							}

							for len(authorizedKeysBytes) > 0 {
								k, _, _, rest, err := gossh.ParseAuthorizedKey(authorizedKeysBytes)
								if err != nil {
									return false
								}

								if ssh.KeysEqual(k, key) {
									return true
								}

								authorizedKeysBytes = rest
							}
						}

						return false
					}),
					sftp.SSHOption(handler),
					wish.WithMiddleware(func(next ssh.Handler) ssh.Handler {
						return func(sess ssh.Session) {
							if sess.User() == "_" {
								rootCmd := NewCmdRoot()
								args := make([]string, 0)
								args = append(args, sess.Command()...)
								rootCmd.SetArgs(args)
								rootCmd.SetIn(sess)
								rootCmd.SetOut(sess)
								rootCmd.SetErr(sess.Stderr())
								rootCmd.CompletionOptions.DisableDefaultCmd = true
								if err := rootCmd.Execute(); err != nil {
									_ = sess.Exit(1)
									return
								}

								return
							}

							a, err := app.LoadApp(sess.User(), k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), sess.User()))
							if err != nil {
								fmt.Fprintf(sess, "failed to load app: %v\n", err)
								return
							}

							wk := worker.NewWorker(a)
							command, err := wk.Command(sess.Context(), sess.Command()...)
							if err != nil {
								fmt.Fprintf(sess, "failed to get command: %v\n", err)
								return
							}

							command.Stdout = sess
							command.Stderr = sess.Stderr()
							stdin, err := command.StdinPipe()
							if err != nil {
								fmt.Fprintf(sess, "failed to get stdin: %v\n", err)
								return
							}

							go func() {
								defer stdin.Close()
								io.Copy(stdin, sess)
							}()

							if err := command.Run(); err != nil {
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
						rsync.Middleware(handler),
						scp.Middleware(handler),
					),
				)

				if err != nil {
					return fmt.Errorf("failed to create wish server: %v", err)
				}

				if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
					fmt.Fprintf(cmd.ErrOrStderr(), "failed to start wish server: %v\n", err)
				}
			}

			// sigint handling
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.httpAddr, "http-addr", "", "address to listen on")
	cmd.Flags().StringVar(&flags.sshAddr, "ssh-addr", "", "address to listen on for ssh/sftp")
	cmd.Flags().StringVar(&flags.sshHostKey, "ssh-host-key", fmt.Sprintf("%s/.ssh/smallweb", os.Getenv("HOME")), "ssh host key")
	cmd.Flags().StringVar(&flags.tlsCert, "tls-cert", "", "tls certificate file")
	cmd.Flags().StringVar(&flags.tlsKey, "tls-key", "", "tls key file")
	cmd.Flags().BoolVar(&flags.cron, "cron", false, "enable cron jobs")
	cmd.Flags().StringVar(&flags.acmdnsCreds, "acmedns-credentials", "", "acme dns credentials")

	cmd.MarkFlagsRequiredTogether("tls-cert", "tls-key")
	cmd.MarkFlagsMutuallyExclusive("acmedns-credentials", "http-addr")
	cmd.MarkFlagsMutuallyExclusive("acmedns-credentials", "tls-cert")
	cmd.MarkFlagsMutuallyExclusive("acmedns-credentials", "tls-key")

	return cmd
}

func getListener(addr, cert, key string) (net.Listener, error) {
	var config *tls.Config
	if cert != "" && key != "" {
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, fmt.Errorf("failed to load cert: %v", err)
		}

		config = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

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
	watcher *watcher.Watcher
	logger  *slog.Logger
	mu      sync.Mutex
	workers map[string]*worker.Worker
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	appname, redirect, ok := lookupApp(r.Host, k.String("domain"), k.StringMap("customDomains"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("No app found for host %s", r.Host)))
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
}

func lookupApp(host string, domain string, customDomains map[string]string) (app string, redirect bool, found bool) {
	// check exact matches first
	for key, value := range customDomains {
		if value == "*" {
			continue
		}

		if key == host {
			return value, false, true
		}
	}

	if host == domain {
		return "www", true, true
	}

	// check for subdomains
	for key, value := range customDomains {
		if value != "*" {
			continue
		}

		if key == host {
			return "www", true, true
		}

		if strings.HasSuffix(host, "."+key) {
			return strings.TrimSuffix(host, "."+key), false, true
		}
	}

	if strings.HasSuffix(host, "."+domain) {
		return strings.TrimSuffix(host, "."+domain), false, true
	}

	return "", false, false
}

func (me *Handler) GetWorker(appname, rootDir, domain string) (*worker.Worker, error) {
	if wk, ok := me.workers[appname]; ok && wk.IsRunning() && me.watcher.GetAppMtime(appname).Before(wk.StartedAt) {
		return wk, nil
	}

	me.mu.Lock()
	defer me.mu.Unlock()

	a, err := app.LoadApp(appname, rootDir, domain, slices.Contains(k.Strings("adminApps"), appname))
	if err != nil {
		return nil, fmt.Errorf("failed to load app: %w", err)
	}

	wk := worker.NewWorker(a)

	wk.Logger = me.logger
	if err := wk.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	me.workers[appname] = wk
	return wk, nil
}
