package cmd

import (
	"context"
	"database/sql"
	"fmt"
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

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

func NewCmdUp() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if k.String("domain") == "" {
				return fmt.Errorf("domain cannot be empty")
			}

			db, err := database.OpenDB(filepath.Join(DataDir(), "smallweb.db"))
			if err != nil {
				return fmt.Errorf("failed to open database: %v", err)
			}

			httpWriter := utils.NewMultiWriter()
			cronWriter := utils.NewMultiWriter()
			consoleWriter := utils.NewMultiWriter()

			consoleLogger := slog.New(slog.NewJSONHandler(consoleWriter, nil))

			apiHandler := api.NewHandler(k, httpWriter, cronWriter, consoleWriter)
			appHandler := &AppHandler{apiServer: apiHandler, db: db, logger: consoleLogger}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rootDir := k.String("dir")

				if r.Host == k.String("domain") {
					// if we are on the apex domain and www exists, redirect to www
					if _, err := os.Stat(filepath.Join(rootDir, "www")); err == nil {
						target := r.URL
						target.Scheme = r.Header.Get("X-Forwarded-Proto")
						if target.Scheme == "" {
							target.Scheme = "http"
						}

						target.Host = "www." + k.String("domain")
						http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
						return
					}

					w.WriteHeader(http.StatusNotFound)
					return
				}

				if !strings.HasSuffix(r.Host, "."+k.String("domain")) {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				appname := strings.TrimSuffix(r.Host, "."+k.String("domain"))
				a, err := app.LoadApp(filepath.Join(rootDir, appname), k.String("domain"))
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				appHandler.ServeApp(w, r, a)
			})

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			cronLogger := slog.New(slog.NewJSONHandler(cronWriter, nil))
			c.AddFunc("* * * * *", func() {
				rounded := time.Now().Truncate(time.Minute)
				rootDir := k.String("dir")
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

						wk := worker.NewWorker(a, consoleLogger)

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

			fmt.Fprintf(os.Stderr, "Serving *.%s from %s on %s\n", k.String("domain"), k.String("dir"), k.String("addr"))
			httpLogger := utils.NewLogger(httpWriter)
			server := http.Server{
				Handler: httpLogger.Middleware(handler),
			}

			addr := k.String("addr")
			var ln net.Listener
			if strings.HasPrefix(addr, "unix/") {
				socketPath := strings.TrimPrefix(addr, "unix/")
				net.Listen("unix", utils.ExpandTilde(socketPath))
			} else {
				net.Listen("tcp", addr)
			}

			go server.Serve(ln)

			// start api server on unix socket
			apiServer := http.Server{
				Handler: apiHandler,
			}

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

				if err := apiServer.Serve(listener); err != nil && err != http.ErrServerClosed {
					log.Fatal(err)
				}
			}()

			// sigint handling
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			log.Println("Shutting down server...")
			apiServer.Shutdown(context.Background())
			c.Stop()
			return nil
		},
	}

	return cmd
}

type AppHandler struct {
	apiServer http.Handler
	db        *sql.DB
	logger    *slog.Logger
}

func (me *AppHandler) ServeApp(w http.ResponseWriter, r *http.Request, a app.App) {
	var handler http.Handler
	if a.Entrypoint() == "smallweb:api" {
		handler = me.apiServer
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
		handler = worker.NewWorker(a, me.logger)
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
		authMiddleware := auth.Middleware(me.db, k.String("email"), a.Name)
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
