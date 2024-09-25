package cmd

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	"github.com/gobwas/glob"
	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/auth"
	"github.com/pomdtr/smallweb/term"
	"golang.org/x/net/webdav"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}

	return nil, nil, fmt.Errorf("Hijack not supported")
}

func requestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{w, http.StatusOK}
			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logger.LogAttrs(context.Background(), slog.LevelInfo, "Request completed",
				slog.String("method", r.Method),
				slog.String("host", r.Host),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Duration("duration", duration),
			)
		})
	}
}

func NewCmdUp(db *sql.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
			rootDir := utils.ExpandTilde(k.String("dir"))
			baseDomain := k.String("domain")

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

			var webdavHandler http.Handler = &webdav.Handler{
				FileSystem: webdav.Dir(rootDir),
				LockSystem: webdav.NewMemLS(),
				Prefix:     "/webdav",
			}

			apiServer := api.NewServer(rootDir, baseDomain)
			apiHandler := api.Handler(&apiServer)

			addr := fmt.Sprintf("%s:%d", k.String("host"), port)
			loggerMiddleware := requestLogger(logger)
			server := http.Server{
				Addr: addr,
				Handler: loggerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Host == baseDomain {
						target := r.URL
						target.Scheme = "https"
						target.Host = "www." + baseDomain
						http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
						return
					}

					var appName string
					if a, ok := k.StringMap("custom-domains")[r.Host]; ok {
						appName = a
					} else {
						if !strings.HasSuffix(r.Host, fmt.Sprintf(".%s", baseDomain)) {
							w.WriteHeader(http.StatusNotFound)
							return
						}

						appName = strings.TrimSuffix(r.Host, fmt.Sprintf(".%s", baseDomain))
					}

					a, err := app.LoadApp(filepath.Join(rootDir, appName), k.String("domain"))
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					var handler http.Handler
					if a.Entrypoint() == "smallweb:api" {
						handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if r.URL.Path == "/openapi.json" {
								w.Header().Set("Content-Type", "text/yaml")
								encoder := json.NewEncoder(w)
								encoder.SetIndent("", "  ")
								encoder.Encode(api.Document)
								return
							}

							if strings.HasPrefix(r.URL.Path, "/webdav") {
								webdavHandler.ServeHTTP(w, r)
								return
							}

							apiHandler.ServeHTTP(w, r)
						})
					} else if a.Entrypoint() == "smallweb:term" {
						handler = term.NewHandler(k.String("shell"), rootDir)
					} else if !strings.HasPrefix(a.Entrypoint(), "smallweb:") {
						wk := worker.NewWorker(a, k.StringMap("env"))
						if err := wk.StartServer(); err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						defer wk.StopServer()
						handler = wk
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
						authMiddleware := auth.Middleware(db, k.String("email"))
						handler = authMiddleware(handler)
					}

					handler.ServeHTTP(w, r)
				})),
			}

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			c.AddFunc("* * * * *", func() {
				rootDir := utils.ExpandTilde(k.String("dir"))
				rounded := time.Now().Truncate(time.Minute)
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

						wk := worker.NewWorker(a, k.StringMap("env"))

						command, err := wk.Command(job.Args...)
						if err != nil {
							fmt.Println(err)
							continue
						}

						if err := command.Run(); err != nil {
							fmt.Println(err)
						}
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
			return server.ListenAndServe()
		},
	}

	return cmd
}
