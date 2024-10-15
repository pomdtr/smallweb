package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	"github.com/gobwas/glob"
	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/auth"
	"github.com/pomdtr/smallweb/database"
	"github.com/pomdtr/smallweb/docs"
	"golang.org/x/crypto/ssh"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

const (
	initialBackoff       = 5 * time.Second // Starting wait time between retries
	maxBackoff           = 1 * time.Minute // Maximum wait time between retries
	sshKeepAliveInterval = 10 * time.Second
	tcpKeepAliveInterval = 10 * time.Second
)

func NewCmdUp() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.OpenDB(filepath.Join(DataDir(), "smallweb.db"))
			if err != nil {
				return fmt.Errorf("failed to open database: %v", err)
			}

			httpWriter := utils.NewMultiWriter()
			cronWriter := utils.NewMultiWriter()
			consoleWriter := utils.NewMultiWriter()

			httpLogger := utils.NewHTTPLogger(httpWriter)
			consoleLogger := slog.New(slog.NewJSONHandler(consoleWriter, nil))

			apiHandler := api.NewHandler(k, httpWriter, cronWriter, consoleWriter)
			handler := &HttpHandler{
				db:         db,
				apiHandler: apiHandler,
				logger:     consoleLogger,
			}

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			cronLogger := slog.New(slog.NewJSONHandler(cronWriter, nil))
			c.AddFunc("* * * * *", func() {
				rounded := time.Now().Truncate(time.Minute)
				rootDir := utils.ExpandTilde(k.String("dir"))
				apps, err := app.ListApps(rootDir)
				if err != nil {
					fmt.Println(err)
				}

				for _, name := range apps {
					a, err := app.LoadApp(filepath.Join(rootDir, name), k.String("domain"))
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

						wk := worker.NewWorker(a, k.StringMap("env"), consoleLogger)

						command, err := wk.Command(job.Args...)
						if err != nil {
							fmt.Println(err)
							continue
						}
						command.Stdout = os.Stdout
						command.Stderr = os.Stderr

						t1 := time.Now()
						var exitCode int
						if err := command.Run(); err != nil {
							if exitError, ok := err.(*exec.ExitError); ok {
								exitCode = exitError.ExitCode()
							}
						}
						duration := time.Since(t1)

						cronLogger.LogAttrs(
							context.Background(),
							slog.LevelInfo,
							fmt.Sprintf("Exit Code: %d", exitCode),
							slog.String("type", "cron"),
							slog.String("id", fmt.Sprintf("%s:%s", a.Name, job.Name)),
							slog.String("app", a.Name),
							slog.String("job", job.Name),
							slog.String("schedule", job.Schedule),
							slog.Any("args", job.Args),
							slog.Int("exit_code", exitCode),
							slog.Int64("duration", duration.Milliseconds()),
						)
					}
				}
			})

			go c.Start()

			go func() {
				socketPath := api.SocketPath(k.String("domain"))
				if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
					log.Fatal(err)
				}

				if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
					log.Fatal(err)
				}
				defer os.Remove(socketPath)

				listener, err := net.Listen("unix", socketPath)
				if err != nil {
					log.Fatal(err)
				}

				apiServer := http.Server{Handler: apiHandler}
				if err := apiServer.Serve(listener); err != nil && err != http.ErrServerClosed {
					log.Fatal(err)
				}
			}()

			if proxy := k.String("proxy"); proxy != "" {
				go func() {
					backoff := initialBackoff
					for {
						fmt.Fprintln(os.Stderr, "Connecting to proxy...")
						conn, err := net.DialTimeout("tcp", proxy, 5*time.Second)
						if err != nil {
							fmt.Fprintf(os.Stderr, "failed to dial SSH: %v, retrying in %v...\n", err, backoff)
							time.Sleep(backoff)
							backoff = backoff * 2 // Exponential backoff
							if backoff > maxBackoff {
								backoff = maxBackoff
							}
							continue
						}

						// Reset backoff on successful connection
						backoff = initialBackoff

						// Enable TCP keep-alive on the connection
						tcpConn, ok := conn.(*net.TCPConn)
						if !ok {
							fmt.Fprintf(os.Stderr, "failed to cast to TCPConn, retrying...\n")
							conn.Close()
							continue
						}

						if err := tcpConn.SetKeepAlive(true); err != nil {
							fmt.Fprintf(os.Stderr, "failed to enable keep-alive: %v, retrying...\n", err)
							conn.Close()
							continue
						}

						if err := tcpConn.SetKeepAlivePeriod(tcpKeepAliveInterval); err != nil {
							fmt.Fprintf(os.Stderr, "failed to set keep-alive period: %v, retrying...\n", err)
							conn.Close()
							continue
						}

						c, chans, reqs, err := ssh.NewClientConn(conn, proxy, &ssh.ClientConfig{
							Timeout:         5 * time.Second,
							HostKeyCallback: ssh.InsecureIgnoreHostKey(),
						})
						if err != nil {
							fmt.Fprintf(os.Stderr, "failed to create SSH client: %v, retrying...\n", err)
							conn.Close()
							continue
						}

						client := ssh.NewClient(c, chans, reqs)
						fmt.Fprintln(os.Stderr, "Connected to proxy")

						go func() {
							ticker := time.NewTicker(sshKeepAliveInterval)
							defer ticker.Stop() // Ensure the ticker is stopped when the goroutine exits

							done := make(chan struct{}) // Channel to signal when connection is closed

							go func() {
								err := client.Wait() // Wait for the client to disconnect (e.g., by the server)
								if err != nil {
									fmt.Fprintf(os.Stderr, "SSH client disconnected: %v\n", err)
								}
								close(done) // Notify the keepalive goroutine to stop
							}()

							for {
								select {
								case <-ticker.C:
									_, _, err := client.SendRequest("keepalive", true, nil)
									if err != nil {
										fmt.Fprintf(os.Stderr, "failed to send keepalive: %v, closing client and retrying...\n", err)
										client.Close() // Close the client to trigger reconnection
										return         // Exit the keepalive goroutine
									}
								case <-done: // Connection closed (by server or client)
									return // Exit the keepalive goroutine
								}
							}
						}()

						channels := client.HandleChannelOpen("forwarded-tcpip")
						if _, _, err := client.SendRequest("tcpip-forward", true, ssh.Marshal(&remoteForwardRequest{
							BindAddr: k.String("domain"),
							BindPort: 80,
						})); err != nil {
							fmt.Fprintf(os.Stderr, "failed to request remote forward: %v, retrying...\n", err)
							client.Close() // Close and retry connection
							continue
						}

						for channel := range channels {
							ch, reqs, err := channel.Accept()
							if err != nil {
								continue
							}
							go ssh.DiscardRequests(reqs)

							c, err := net.Dial("tcp", fmt.Sprintf(":%d", k.Int("port")))
							if err != nil {
								ch.Close()
								continue
							}
							go func() {
								defer c.Close()
								defer ch.Close()
								io.Copy(c, ch)
							}()

							go func() {
								defer ch.Close()
								defer c.Close()
								io.Copy(ch, c)
							}()
						}
					}
				}()

			}

			server := http.Server{
				Addr:    fmt.Sprintf(":%d", k.Int("port")),
				Handler: httpLogger.Middleware(handler),
			}

			fmt.Fprintf(os.Stderr, "Serving *.%s domains on port %d...\n", k.String("domain"), k.Int("port"))
			go server.ListenAndServe()

			// sigint handling
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			log.Println("Shutting down server...")
			server.Shutdown(context.Background())
			c.Stop()
			return nil
		},
	}

	return cmd
}

type HttpHandler struct {
	db         *sql.DB
	apiHandler http.Handler
	logger     *slog.Logger
}

func (h *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rootDir := utils.ExpandTilde(k.String("dir"))

	appname := func() string {
		if r.Host == k.String("domain") {
			if stat, err := os.Stat(filepath.Join(rootDir, "@")); err == nil && stat.IsDir() {
				return "@"
			}

			return ""
		}

		if appname := strings.TrimSuffix(r.Host, fmt.Sprintf(".%s", k.String("domain"))); utils.FileExists(filepath.Join(rootDir, appname)) {
			return appname
		}

		return ""
	}()

	if appname == "" {
		if r.Host == k.String("domain") {
			// if we are on the apex domain and www exists, redirect to www
			if stat, err := os.Stat(filepath.Join(rootDir, "www")); err == nil && stat.IsDir() {
				target := r.URL
				target.Scheme = "https"
				target.Host = "www." + k.String("domain")
				http.Redirect(w, r, target.String(), http.StatusFound)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Host == fmt.Sprintf("www.%s", k.String("domain")) {
			// if we are on the www domain and apex exists, redirect to apex
			if stat, err := os.Stat(filepath.Join(rootDir, "@")); err == nil && stat.IsDir() {
				target := r.URL
				target.Scheme = "https"
				target.Host = k.String("domain")
				http.Redirect(w, r, target.String(), http.StatusFound)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		return
	}

	a, err := app.LoadApp(filepath.Join(rootDir, appname), k.String("domain"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var handler http.Handler
	if a.Entrypoint() == "smallweb:api" {
		handler = h.apiHandler
	} else if a.Entrypoint() == "smallweb:docs" {
		handler = docs.Handler
	} else if a.Entrypoint() == "smallweb:static" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if strings.HasPrefix(filepath.Base(r.URL.Path), ".") {
				http.Error(w, "file not found", http.StatusNotFound)
				return
			}

			p := filepath.Join(a.Root(), r.URL.Path)
			transformOptions := esbuild.TransformOptions{
				Target:       esbuild.ESNext,
				Format:       esbuild.FormatESModule,
				MinifySyntax: false,
				Sourcemap:    esbuild.SourceMapNone,
			}

			switch path.Ext(r.URL.Path) {
			case ".ts":
				transformOptions.Loader = esbuild.LoaderTS
				code, err := transpile(p, transformOptions)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				w.Header().Set("Content-Type", "application/javascript")
				w.Write(code)
				return
			case ".jsx":
				transformOptions.Loader = esbuild.LoaderJSX
				transformOptions.JSX = esbuild.JSXAutomatic
				code, err := transpile(p, transformOptions)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				w.Header().Set("Content-Type", "application/javascript")
				w.Write(code)
				return
			case ".tsx":
				transformOptions.Loader = esbuild.LoaderTSX
				transformOptions.JSX = esbuild.JSXAutomatic
				code, err := transpile(p, transformOptions)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				w.Header().Set("Content-Type", "application/javascript")
				w.Write(code)
				return
			default:
				if utils.FileExists(p) {
					http.ServeFile(w, r, p)
					return
				}

				if utils.FileExists(p + ".html") {
					http.ServeFile(w, r, p+".html")
					return
				}

				http.Error(w, "file not found", http.StatusNotFound)
				return
			}
		})
	} else if !strings.HasPrefix(a.Entrypoint(), "smallweb:") {
		handler = worker.NewWorker(a, k.StringMap("env"), h.logger)
	} else {
		http.Error(w, "invalid entrypoint", http.StatusInternalServerError)
		return
	}

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

	if isPrivateRoute || strings.HasPrefix(r.URL.Path, "/_auth") {
		authMiddleware := auth.Middleware(h.db, k.String("email"), appname)
		handler = authMiddleware(handler)
	}

	handler.ServeHTTP(w, r)
}

func transpile(p string, options esbuild.TransformOptions) ([]byte, error) {
	content, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	result := esbuild.Transform(string(content), options)

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf(result.Errors[0].Text)
	}

	return result.Code, nil
}
