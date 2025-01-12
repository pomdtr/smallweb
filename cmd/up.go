package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
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
	"syscall"
	"unsafe"

	_ "embed"

	"github.com/caddyserver/certmagic"
	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/ssh"
	"github.com/creack/pty"
	"github.com/pkg/sftp"
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
		addr        string
		sshAddr     string
		sshHostKey  string
		onDemandTLS bool
		cert        string
		key         string
	}

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		PreRunE: requireDomain,
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

			if flags.onDemandTLS {
				certmagic.Default.OnDemand = &certmagic.OnDemandConfig{
					DecisionFunc: func(ctx context.Context, name string) error {
						appname, _, ok := lookupApp(name, k.String("domain"), k.StringMap("customDomains"))
						if !ok {
							return fmt.Errorf("failed to lookup app: %v", err)
						}

						if _, err := app.NewApp(appname, k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), appname)); err != nil {
							return fmt.Errorf("failed to load app: %v", err)
						}

						return nil
					},
				}

				fmt.Fprintf(os.Stderr, "Serving *.%s from %s with on-demand TLS...\n", k.String("domain"), utils.AddTilde(k.String("dir")))
				go certmagic.HTTPS(nil, handler)
			} else {
				addr := flags.addr
				if addr == "" {
					if flags.cert != "" || flags.key != "" {
						addr = "0.0.0.0:443"
					} else {
						addr = "localhost:7777"
					}
				}

				listener, err := getListener(addr, flags.cert, flags.key)
				if err != nil {
					return fmt.Errorf("failed to get listener: %v", err)
				}

				fmt.Fprintf(os.Stderr, "Serving *.%s from %s on %s...\n", k.String("domain"), utils.AddTilde(k.String("dir")), addr)
				go http.Serve(listener, handler)
			}

			if flags.cron {
				fmt.Fprintln(os.Stderr, "Starting cron jobs...")
				crons := CronRunner()
				crons.Start()
				defer crons.Stop()
			}

			if flags.sshAddr != "" {
				server := ssh.Server{
					PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
						for _, authorizedKeysPath := range []string{
							filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys"),
							filepath.Join(k.String("dir"), ".smallweb", "authorized_keys"),
						} {
							ok, err := validatePublicKey(authorizedKeysPath, key)
							if err != nil {
								if errors.Is(err, os.ErrNotExist) {
									continue
								}

								fmt.Fprintf(os.Stderr, "%s\n", err)
								continue
							}

							if ok {
								return true
							}

						}

						return false
					},
					SubsystemHandlers: map[string]ssh.SubsystemHandler{
						"sftp": func(sess ssh.Session) {
							var workDir string
							if sess.User() == "_" {
								workDir = k.String("dir")
							} else {
								app, err := app.NewApp(sess.User(), k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), sess.User()))
								if err != nil {
									fmt.Fprintln(sess, "failed to load app:", err)
									return
								}

								workDir = filepath.Join(app.Root(), "data")
								if _, err := os.Stat(workDir); err != nil {
									fmt.Fprintln(sess, "failed to get app data directory")
									return
								}
							}

							server, err := sftp.NewServer(
								sess,
								sftp.WithServerWorkingDirectory(workDir),
							)

							if err != nil {
								log.Printf("sftp server init error: %s\n", err)
								return
							}
							if err := server.Serve(); err == io.EOF {
								server.Close()
								fmt.Println("sftp client exited session.")
							} else if err != nil {
								fmt.Println("sftp server completed with error:", err)
							}
						},
					},
					Handler: func(sess ssh.Session) {
						var cmd *exec.Cmd
						if sess.User() == "_" {
							execPath, err := os.Executable()
							if err != nil {
								fmt.Fprintf(sess, "failed to get executable path: %v\n", err)
								return
							}
							cmd = exec.Command(execPath, sess.Command()...)
							cmd.Env = os.Environ()
						} else {
							app, err := app.NewApp(sess.User(), k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), sess.User()))
							if err != nil {
								fmt.Fprintf(sess, "failed to load app: %v\n", err)
								return
							}

							wk := worker.NewWorker(app, k.String("dir"), k.String("domain"))
							command, err := wk.Command(sess.Context(), sess.Command()...)
							if err != nil {
								fmt.Fprintf(sess, "failed to get command: %v\n", err)
								return
							}

							cmd = command
						}

						ptyReq, winCh, isPty := sess.Pty()
						if isPty {
							cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
							f, err := pty.Start(cmd)
							if err != nil {
								panic(err)
							}
							go func() {
								for win := range winCh {
									syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
										uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(win.Height), uint16(win.Width), 0, 0})))
								}
							}()
							go func() {
								io.Copy(f, sess)
							}()
							io.Copy(sess, f)
							cmd.Wait()
							return
						} else {
							stdin, err := cmd.StdinPipe()
							if err != nil {
								fmt.Fprintf(sess, "failed to get stdin pipe: %v", err)
								return
							}

							go func() {
								io.Copy(stdin, sess)
							}()

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
						}
					},
				}

				hostKeyPath := utils.ExpandTilde(flags.sshHostKey)
				if !cmd.Flags().Changed("ssh-host-key") && !utils.FileExists(utils.ExpandTilde(hostKeyPath)) {
					kp, err := keygen.New(hostKeyPath, keygen.WithKeyType(keygen.Ed25519))
					if err != nil {
						return fmt.Errorf("failed to generate ssh key: %v", err)
					}

					if err := kp.WriteKeys(); err != nil {
						return fmt.Errorf("failed to write ssh key: %v", err)
					}
				}

				server.SetOption(ssh.HostKeyFile(hostKeyPath))

				listener, err := getListener(flags.sshAddr, "", "")
				if err != nil {
					return fmt.Errorf("failed to get ssh listener: %v", err)
				}

				fmt.Fprintf(os.Stderr, "Starting SSH server on %s...\n", flags.sshAddr)
				go server.Serve(listener)
				defer server.Close()
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
	cmd.Flags().StringVar(&flags.sshHostKey, "ssh-host-key", "~/.ssh/smallweb", "ssh host key")
	cmd.Flags().BoolVar(&flags.onDemandTLS, "on-demand-tls", false, "enable on-demand TLS")
	cmd.Flags().StringVar(&flags.cert, "cert", "", "tls certificate file")
	cmd.Flags().StringVar(&flags.key, "key", "", "key file")
	cmd.Flags().BoolVar(&flags.cron, "cron", false, "enable cron jobs")

	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "cert")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "key")
	cmd.MarkFlagsMutuallyExclusive("on-demand-tls", "addr")

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
			return tls.Listen("unix", utils.ExpandTilde(socketPath), config)
		}

		return net.Listen("unix", utils.ExpandTilde(socketPath))
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
		w.Write([]byte(fmt.Sprintf("No app found for host %s", r.Host)))
		w.WriteHeader(http.StatusNotFound)
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
			w.Write([]byte(fmt.Sprintf("No app found for host %s", r.Host)))
			w.WriteHeader(http.StatusNotFound)
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

	a, err := app.NewApp(appname, rootDir, domain, slices.Contains(k.Strings("adminApps"), appname))
	if err != nil {
		return nil, fmt.Errorf("failed to load app: %w", err)
	}

	wk := worker.NewWorker(a, k.String("dir"), domain)

	wk.Logger = me.logger
	if err := wk.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	me.workers[appname] = wk
	return wk, nil
}

func requireDomain(cmd *cobra.Command, args []string) error {
	if k.String("domain") == "" {
		return errors.New("missing domain")
	}
	return nil
}

func validatePublicKey(authorizedKeysPath string, pubKey ssh.PublicKey) (bool, error) {
	authorizedKeysBytes, err := os.ReadFile(authorizedKeysPath)
	if err != nil {
		return false, err
	}

	for len(authorizedKeysBytes) > 0 {
		k, _, _, rest, err := gossh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return false, err
		}

		if ssh.KeysEqual(k, pubKey) {
			return true, nil
		}

		authorizedKeysBytes = rest
	}

	return false, nil
}
