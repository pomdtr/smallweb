package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
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

	"github.com/caddyserver/certmagic"
	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/creack/pty"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/sftp"
	"github.com/pomdtr/smallweb/watcher"
	gossh "golang.org/x/crypto/ssh"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	var flags struct {
		enableCrons bool
		addr        string
		apiAddr     string
		sshAddr     string
		sshHostKey  string
		tlsCert     string
		tlsKey      string
		onDemandTLS bool
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

			oldCronFlag, _ := cmd.Flags().GetBool("cron")
			if oldCronFlag {
				flags.enableCrons = true
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if k.String("dir") == "" {
				return fmt.Errorf("dir cannot be empty")
			}

			if k.String("domain") == "" {
				return fmt.Errorf("domain cannot be empty")
			}

			watcher, err := watcher.NewWatcher(k.String("dir"), func() {
				fileProvider := file.Provider(utils.FindConfigPath(k.String("dir")))
				flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)

				k = koanf.New(".")
				_ = k.Load(fileProvider, utils.ConfigParser())
				_ = k.Load(envProvider, nil)
				_ = k.Load(flagProvider, nil)
			})
			if err != nil {
				return fmt.Errorf("failed to create watcher: %v", err)
			}

			go watcher.Start()
			defer watcher.Stop()

			handler := &Handler{
				watcher: watcher,
				workers: make(map[string]*worker.Worker),
			}

			if flags.onDemandTLS {
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
				fmt.Fprintf(cmd.ErrOrStderr(), "Serving *.%s from %s on %s...\n", k.String("domain"), utils.AddTilde(k.String("dir")), ":443")
				go certmagic.HTTPS(nil, handler)
			} else if flags.tlsCert != "" && flags.tlsKey != "" {
				cert, err := tls.LoadX509KeyPair(flags.tlsCert, flags.tlsKey)
				if err != nil {
					return fmt.Errorf("failed to load tls certificate: %v", err)
				}

				tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
				tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)

				addr := flags.addr
				if addr == "" {
					addr = ":443"
				}

				ln, err := getListener(addr, tlsConfig)
				if err != nil {
					return fmt.Errorf("failed to get listener: %v", err)
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Serving *.%s from %s on %s...\n", k.String("domain"), utils.AddTilde(k.String("dir")), addr)
				go http.Serve(ln, handler)
			} else {
				addr := flags.addr
				if addr == "" {
					addr = ":7777"
				}

				ln, err := getListener(addr, nil)
				if err != nil {
					return fmt.Errorf("failed to get listener: %v", err)
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Serving *.%s from %s on %s...\n", k.String("domain"), utils.AddTilde(k.String("dir")), addr)
				go http.Serve(ln, handler)
			}

			if flags.enableCrons {
				fmt.Fprintln(cmd.ErrOrStderr(), "Starting cron jobs...")
				crons := CronRunner(cmd.OutOrStdout(), cmd.ErrOrStderr())
				crons.Start()
				defer crons.Stop()
			}

			if flags.apiAddr != "" {
				mux := http.NewServeMux()

				mux.HandleFunc("GET /caddy/ask", func(w http.ResponseWriter, r *http.Request) {
					domain := r.URL.Query().Get("domain")
					if domain == "" {
						http.Error(w, "domain parameter is required", http.StatusBadRequest)
						return
					}

					_, _, found := lookupApp(domain)
					if !found {
						http.Error(w, "app not found", http.StatusNotFound)
						return
					}

					w.Write([]byte("ok"))
				})

				ln, err := getListener(flags.apiAddr, nil)
				if err != nil {
					return fmt.Errorf("failed to get listener for api: %v", err)
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Starting api server on %s...\n", flags.apiAddr)
				go http.Serve(ln, mux)
			}

			if flags.sshAddr != "" {
				hostKey := flags.sshHostKey
				if hostKey == "" {
					homeDir, err := os.UserHomeDir()
					if err != nil {
						return fmt.Errorf("failed to get home directory: %v", err)
					}
					hostKey = filepath.Join(homeDir, ".ssh", "smallweb")
				}

				if !utils.FileExists(hostKey) {
					_, err := keygen.New(hostKey, keygen.WithWrite())
					if err != nil {
						return fmt.Errorf("failed to generate host key: %v", err)
					}
				}

				srv, err := wish.NewServer(
					wish.WithAddress(flags.sshAddr),
					wish.WithHostKeyPath(hostKey),
					wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
						authorizedKeys := k.Strings("authorizedKeys")
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
					wish.WithMiddleware(func(next ssh.Handler) ssh.Handler {
						return func(sess ssh.Session) {
							if sess.User() != "_" {
								a, err := app.LoadApp(sess.User(), k.String("dir"), k.String("domain"), k.Bool(fmt.Sprintf("apps.%s.admin", sess.User())))
								if err != nil {
									fmt.Fprintf(sess, "failed to load app: %v\n", err)
									sess.Exit(1)
									return
								}

								wk := worker.NewWorker(a)
								cmd, err := wk.Command(sess.Context(), sess.Command()...)
								if err != nil {
									fmt.Fprintf(sess, "failed to get command: %v\n", err)
									sess.Exit(1)
									return
								}

								cmd.Stdout = sess
								cmd.Stderr = sess.Stderr()

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

								return
							}

							execPath, err := os.Executable()
							if err != nil {
								fmt.Fprintf(sess.Stderr(), "failed to get executable path: %v\n", err)
								sess.Exit(1)
								return
							}

							cmd := exec.Command(execPath, "--dir", k.String("dir"), "--domain", k.String("domain"))
							cmd.Args = append(cmd.Args, sess.Command()...)
							cmd.Env = os.Environ()
							cmd.Env = append(cmd.Env, "SMALLWEB_DISABLE_PLUGINS=true")

							ptyReq, _, isPty := sess.Pty()
							if isPty {
								cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
								f, err := pty.Start(cmd)
								if err != nil {
									fmt.Fprintf(sess, "failed to start pty: %v\n", err)
									sess.Exit(1)
									return
								}

								go func() {
									io.Copy(f, sess)
								}()
								io.Copy(sess, f)

								if err := cmd.Wait(); err != nil {
									var exitErr *exec.ExitError
									if errors.As(err, &exitErr) {
										sess.Exit(exitErr.ExitCode())
									}
									sess.Exit(1)
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
					}),
				)

				if err != nil {
					return fmt.Errorf("failed to create ssh server: %v", err)
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Starting ssh server on %s...\n", flags.sshAddr)
				if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
					fmt.Fprintf(cmd.ErrOrStderr(), "failed to start ssh server: %v\n", err)
				}
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
	cmd.Flags().StringVar(&flags.sshHostKey, "ssh-host-key", "", "ssh host key")
	cmd.Flags().StringVar(&flags.tlsCert, "tls-cert", "", "tls certificate file")
	cmd.Flags().StringVar(&flags.tlsKey, "tls-key", "", "tls key file")
	cmd.Flags().StringVar(&flags.apiAddr, "api-addr", "", "address to listen on for api")
	cmd.Flags().BoolVar(&flags.enableCrons, "enable-crons", false, "enable cron jobs")
	cmd.Flags().Bool("cron", false, "enable cron jobs")
	cmd.Flags().MarkDeprecated("cron", "use --enable-crons instead")
	cmd.Flags().BoolVar(&flags.onDemandTLS, "on-demand-tls", false, "enable on-demand tls")

	cmd.MarkFlagsRequiredTogether("tls-cert", "tls-key")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "tls-cert")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "tls-key")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "addr")

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
	watcher *watcher.Watcher
	mu      sync.Mutex
	workers map[string]*worker.Worker
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

func lookupApp(domain string) (app string, redirect bool, found bool) {
	if domain == k.String("domain") {
		return "www", true, true
	}

	for _, app := range k.MapKeys("apps") {
		if slices.Contains(k.Strings(fmt.Sprintf("apps.%s.additionalDomains", app)), domain) {
			return app, false, true
		}
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

func (me *Handler) GetWorker(appname, rootDir, domain string) (*worker.Worker, error) {
	if wk, ok := me.workers[appname]; ok && wk.IsRunning() && me.watcher.GetAppMtime(appname).Before(wk.StartedAt) {
		return wk, nil
	}

	me.mu.Lock()
	defer me.mu.Unlock()

	a, err := app.LoadApp(appname, rootDir, domain, k.Bool(fmt.Sprintf("apps.%s.admin", appname)))
	if err != nil {
		return nil, fmt.Errorf("failed to load app: %w", err)
	}

	wk := worker.NewWorker(a)

	if err := wk.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	me.workers[appname] = wk
	return wk, nil
}
