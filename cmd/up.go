package cmd

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	"github.com/gobwas/glob"
	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/auth"
	"github.com/pomdtr/smallweb/term"

	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

func NewCmdUp(db *sql.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			multiwriter := utils.NewMultiWriter()
			logger := utils.NewLogger(multiwriter)

			apiHandler := api.NewHandler(k, multiwriter)
			addr := fmt.Sprintf("%s:%d", k.String("host"), port)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					handler = apiHandler
				} else if a.Entrypoint() == "smallweb:terminal" {
					handler = term.NewHandler(k.String("shell"), rootDir)
				} else if a.Entrypoint() == "smallweb:file-server" {
					handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					handler = worker.NewWorker(a, k.StringMap("env"))
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
			})

			if err := os.MkdirAll(filepath.Dir(httpLogFile), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(httpLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			server := http.Server{
				Addr:    addr,
				Handler: logger.HTTPResponseLogger(handler),
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
