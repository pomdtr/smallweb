package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"golang.org/x/net/webdav"
)

func basicAuth(h http.Handler, user, pass string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func NewCmdUp() *cobra.Command {
	var flags struct {
		addr string
		cert string
		key  string
		user string
		pass string
	}

	cmd := &cobra.Command{
		Use:     "up <domain>",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			addr := flags.addr
			if addr == "" {
				if flags.cert != "" || flags.key != "" {
					addr = ":443"
				} else {
					addr = ":7777"
				}
			}

			var webdavHandler http.Handler = &webdav.Handler{
				FileSystem: webdav.Dir(rootDir),
				LockSystem: webdav.NewMemLS(),
			}

			if flags.user != "" || flags.pass != "" {
				webdavHandler = basicAuth(webdavHandler, flags.user, flags.pass)
			}

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
						webdavHandler.ServeHTTP(w, r)
						return
					}

					apps := ListApps()
					for _, app := range apps {
						if r.Host != fmt.Sprintf("%s.%s", app, domain) {
							continue
						}

						wk, err := worker.NewWorker(filepath.Join(rootDir, app))
						if err != nil {
							w.WriteHeader(http.StatusNotFound)
							return
						}

						if err := wk.Start(); err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						if wk.Config.Private {
							handler := basicAuth(wk, flags.user, flags.pass)
							handler.ServeHTTP(w, r)
						} else {
							wk.ServeHTTP(w, r)
						}

						if err := wk.Stop(); err != nil {
							log.Printf("failed to stop worker: %v", err)
							return
						}
						return
					}

					// no app was found
					w.WriteHeader(http.StatusNotFound)
				}),
			}

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			c.AddFunc("* * * * *", func() {
				rounded := time.Now().Truncate(time.Minute)

				apps := ListApps()

				for _, app := range apps {
					w, err := worker.NewWorker(app)
					if err != nil {
						fmt.Println(err)
						continue
					}

					for _, job := range w.Config.Crons {
						sched, err := parser.Parse(job.Schedule)
						if err != nil {
							fmt.Println(err)
							continue
						}

						if sched.Next(rounded.Add(-1*time.Second)) != rounded {
							continue
						}

						go w.Run(job.Args)
					}

				}
			})

			go c.Start()

			if flags.cert != "" || flags.key != "" {
				if flags.cert == "" {
					return fmt.Errorf("TLS certificate file is required")
				}

				if flags.key == "" {
					return fmt.Errorf("TLS key file is required")
				}

				cert, err := tls.LoadX509KeyPair(flags.cert, flags.key)
				if err != nil {
					return fmt.Errorf("failed to load TLS certificate and key: %w", err)
				}

				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS12,
				}

				server.TLSConfig = tlsConfig

				cmd.Printf("Evaluation server listening on %s\n", addr)
				return server.ListenAndServeTLS(flags.cert, flags.key)
			}

			cmd.Printf("Evaluation server listening on %s\n", addr)
			return server.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&flags.addr, "addr", "", "Address to listen on")
	cmd.Flags().StringVar(&flags.cert, "cert", "", "TLS certificate file path")
	cmd.Flags().StringVar(&flags.key, "key", "", "TLS key file path")
	cmd.Flags().StringVar(&flags.user, "user", "", "Basic auth username")
	cmd.Flags().StringVar(&flags.pass, "pass", "", "Basic auth password")

	return cmd
}
