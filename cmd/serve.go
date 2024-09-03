package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/term"
	"github.com/pomdtr/smallweb/utils"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"golang.org/x/net/webdav"
)

func authMiddleware(h http.Handler, username string, password string, tokens []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if ok {
			if u == username && p == password {
				h.ServeHTTP(w, r)
				return
			}

			for _, t := range tokens {
				if u == t {
					h.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		authorization := r.Header.Get("Authorization")
		if authorization != "" {
			token := strings.TrimPrefix(authorization, "Bearer ")
			for _, t := range tokens {
				if token == t {
					h.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Bearer realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"up"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			domain := k.String("domain")
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

			cliHandler, err := term.NewHandler(rootDir, k.String("editor"))
			if err != nil {
				return fmt.Errorf("failed to create cli handler: %w", err)
			}

			addr := fmt.Sprintf("%s:%d", k.String("host"), port)
			server := http.Server{
				Addr: addr,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Host == domain {
						target := r.URL
						target.Scheme = "https"
						target.Host = "www." + domain
						http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
					}

					if r.Host == fmt.Sprintf("webdav.%s", domain) {
						var handler http.Handler = &webdav.Handler{
							FileSystem: webdav.Dir(utils.ExpandTilde(k.String("dir"))),
							LockSystem: webdav.NewMemLS(),
						}
						if k.String("tokens") != "" {
							handler = authMiddleware(handler, k.String("username"), k.String("password"), k.Strings("tokens"))
						}

						handler.ServeHTTP(w, r)
						return
					}

					if r.Host == fmt.Sprintf("cli.%s", domain) {
						var handler http.Handler = cliHandler
						if k.String("tokens") != "" {
							handler = authMiddleware(cliHandler, k.String("username"), k.String("password"), k.Strings("tokens"))
						}

						handler.ServeHTTP(w, r)
						return
					}

					var appDir string
					if strings.HasSuffix(r.Host, fmt.Sprintf(".%s", domain)) {
						appname := strings.TrimSuffix(r.Host, fmt.Sprintf(".%s", domain))
						appDir = filepath.Join(rootDir, appname)
						if !utils.FileExists(appDir) {
							w.WriteHeader(http.StatusNotFound)
							return
						}
					} else {
						for _, appname := range ListApps(rootDir) {
							cnamePath := filepath.Join(rootDir, appname, "CNAME")
							if !utils.FileExists("CNAME") {
								continue
							}

							cnameBytes, err := os.ReadFile(cnamePath)
							if err != nil {
								continue
							}

							if r.Host != string(cnameBytes) {
								continue
							}

							appDir = filepath.Join(rootDir, appname)
						}

						if appDir == "" {
							log.Printf("App not found for %s", r.Host)
							w.WriteHeader(http.StatusNotFound)
							return
						}
					}

					a, err := app.NewApp(appDir, r.Host, k.StringMap("env"))
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					if err := a.Start(); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					defer a.Stop()

					var handler http.Handler = a
					if a.Config.Private {
						handler = authMiddleware(a, k.String("username"), k.String("password"), k.Strings("tokens"))
					}
					handler.ServeHTTP(w, r)
				}),
			}

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			c.AddFunc("* * * * *", func() {
				rootDir := utils.ExpandTilde(k.String("dir"))
				rounded := time.Now().Truncate(time.Minute)
				apps := ListApps(rootDir)

				for _, appname := range apps {
					a, err := app.NewApp(appname, fmt.Sprintf("%s.%s", appname, k.String("domain")), k.StringMap("env"))
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

						go a.Run(job.Args)
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

				certificate, err := tls.LoadX509KeyPair(cert, key)
				if err != nil {
					return fmt.Errorf("failed to load TLS certificate and key: %w", err)
				}

				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{certificate},
					MinVersion:   tls.VersionTLS12,
				}

				server.TLSConfig = tlsConfig

				cmd.Printf("Serving %s from %s on %s\n", k.String("domain"), k.String("dir"), addr)
				return server.ListenAndServeTLS(cert, key)
			}

			cmd.Printf("Serving *.%s from %s on %s\n", k.String("domain"), k.String("dir"), addr)
			return server.ListenAndServe()
		},
	}

	return cmd
}
