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

	"github.com/pomdtr/smallweb/utils"
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
		domain         string
		host           string
		port           int
		cert           string
		key            string
		user           string
		pass           string
		installService bool
	}

	cmd := &cobra.Command{
		Use:     "up <domain>",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.port == 0 {
				if flags.cert != "" || flags.key != "" {
					flags.port = 443
				} else {
					flags.port = 7777
				}
			}

			if flags.installService {
				args := []string{
					"up",
					fmt.Sprintf("--domain=%s", flags.domain),
					fmt.Sprintf("--host=%s", flags.host),
					fmt.Sprintf("--port=%d", flags.port),
				}

				if flags.cert != "" {
					cert, err := filepath.Abs(flags.cert)
					if err != nil {
						return fmt.Errorf("failed to get absolute path of TLS certificate: %w", err)
					}

					args = append(args, fmt.Sprintf("--cert=%s", cert))
				}

				if flags.key != "" {
					key, err := filepath.Abs(flags.key)
					if err != nil {
						return fmt.Errorf("failed to get absolute path of TLS key: %w", err)
					}

					args = append(args, fmt.Sprintf("--key=%s", key))
				}

				if flags.user != "" {
					args = append(args, fmt.Sprintf("--user=%s", flags.user))
				}

				if flags.pass != "" {
					args = append(args, fmt.Sprintf("--pass=%s", flags.pass))
				}

				return InstallService(args)
			}

			addr := fmt.Sprintf("%s:%d", flags.host, flags.port)

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
					if r.Host == flags.domain {
						target := r.URL
						target.Scheme = "https"
						target.Host = "www." + flags.domain
						http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
					}

					if strings.HasSuffix(r.Host, flags.domain) {
						for _, app := range ListApps() {
							cnamePath := filepath.Join(rootDir, app, "CNAME")
							if !utils.FileExists(cnamePath) {
								continue
							}

							cname, err := os.ReadFile(cnamePath)
							if err != nil {
								log.Printf("failed to read CNAME file: %v", err)
								continue
							}

							if strings.TrimSpace(string(cname)) == r.Host {
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
						}
					}

					if r.Host == fmt.Sprintf("webdav.%s", flags.domain) {
						webdavHandler.ServeHTTP(w, r)
						return
					}

					apps := ListApps()
					for _, app := range apps {
						if r.Host != fmt.Sprintf("%s.%s", app, flags.domain) {
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

	cmd.Flags().BoolVar(&flags.installService, "install-service", false, "Install the smallweb evaluation server as a service")
	cmd.Flags().StringVar(&flags.domain, "domain", "localhost", "Domain name")
	cmd.Flags().StringVar(&flags.host, "host", "127.0.0.1", "Server host")
	cmd.Flags().IntVar(&flags.port, "port", 0, "Server port")
	cmd.Flags().StringVar(&flags.cert, "cert", "", "TLS certificate file path")
	cmd.Flags().StringVar(&flags.key, "key", "", "TLS key file path")
	cmd.Flags().StringVar(&flags.user, "user", "", "Basic auth username")
	cmd.Flags().StringVar(&flags.pass, "pass", "", "Basic auth password")

	return cmd
}
