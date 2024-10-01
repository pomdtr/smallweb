package cmd

import (
	"context"
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

	"github.com/adrg/xdg"
	"github.com/gobwas/glob"
	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/auth"
	"github.com/pomdtr/smallweb/database"
	"github.com/pomdtr/smallweb/docs"

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
			db, err := database.OpenDB(filepath.Join(xdg.DataHome, "smallweb", "smallweb.db"))
			if err != nil {
				return fmt.Errorf("failed to open database: %v", err)
			}

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

			httpWriter := utils.NewMultiWriter()
			cronWriter := utils.NewMultiWriter()
			consoleWriter := utils.NewMultiWriter()

			httpLogger := utils.NewLogger(httpWriter)
			consoleLogger := slog.New(slog.NewJSONHandler(consoleWriter, nil))

			apiHandler := api.NewHandler(k, httpWriter, cronWriter, consoleWriter)
			addr := fmt.Sprintf("%s:%d", k.String("host"), port)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Host == k.String("domain") {
					target := r.URL
					target.Scheme = "https"
					target.Host = "www." + k.String("domain")
					http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
					return
				}

				appname := func() string {
					for domainGlob, app := range k.StringMap("customDomains") {
						g := glob.MustCompile(domainGlob)
						if g.Match(r.Host) {
							return app
						}
					}

					if strings.HasSuffix(r.Host, fmt.Sprintf(".%s", k.String("domain"))) {
						return strings.TrimSuffix(r.Host, fmt.Sprintf(".%s", k.String("domain")))
					}

					return ""
				}()

				if appname == "" {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				rootDir := utils.ExpandTilde(k.String("dir"))
				a, err := app.LoadApp(filepath.Join(rootDir, appname), k.String("domain"))
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				var handler http.Handler
				if a.Entrypoint() == "smallweb:api" {
					handler = apiHandler
				} else if a.Entrypoint() == "smallweb:docs" {
					handler = docs.Handler
				} else if a.Entrypoint() == "smallweb:file-server" {
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
					handler = worker.NewWorker(a, k.StringMap("env"), consoleLogger)
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
					authMiddleware := auth.Middleware(db, k.String("email"), appname)
					handler = authMiddleware(handler)
				}

				handler.ServeHTTP(w, r)
			})

			server := http.Server{
				Addr:    addr,
				Handler: httpLogger.Middleware(handler),
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

			if cert != "" || key != "" {
				if cert == "" {
					return fmt.Errorf("TLS certificate file is required")
				}

				if key == "" {
					return fmt.Errorf("TLS key file is required")
				}

				cmd.Printf("Serving %s from %s on %s\n", k.String("domain"), k.String("dir"), addr)
				return server.ListenAndServeTLS(utils.ExpandTilde(cert), utils.ExpandTilde(key))
			}

			cmd.Printf("Serving *.%s from %s on %s\n", k.String("domain"), k.String("dir"), addr)
			go server.ListenAndServe()

			// start api server on unix socket
			apiServer := http.Server{
				Handler: apiHandler,
			}

			go func() {
				if err := os.MkdirAll(filepath.Dir(api.SocketPath), 0755); err != nil {
					log.Fatal(err)
				}

				if err := os.Remove(api.SocketPath); err != nil && !os.IsNotExist(err) {
					log.Fatal(err)
				}
				defer os.Remove(api.SocketPath)

				listener, err := net.Listen("unix", api.SocketPath)
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
			server.Shutdown(context.Background())
			apiServer.Shutdown(context.Background())
			c.Stop()
			return nil
		},
	}

	return cmd
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
