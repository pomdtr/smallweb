package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	var flags struct {
		host          string
		port          int
		tls           bool
		tlsCert       string
		tlsKey        string
		tlsCaCert     string
		tlsClientAuth string
	}

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			port := flags.port
			if flags.port == 0 && flags.tls {
				port = 443
			} else if flags.port == 0 {
				port = 7777
			}
			addr := fmt.Sprintf("%s:%d", flags.host, port)

			server := http.Server{
				Addr: addr,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if worker.Exists(filepath.Join(worker.SMALLWEB_ROOT, r.Host, "www")) {
						http.Redirect(w, r, fmt.Sprintf("https://www.%s", r.Host), http.StatusTemporaryRedirect)
						return
					}

					parts := strings.SplitN(r.Host, ".", 2)
					subdomain, domain := parts[0], parts[1]
					appDir := filepath.Join(worker.SMALLWEB_ROOT, domain, subdomain)
					if !worker.Exists(filepath.Join(worker.SMALLWEB_ROOT, domain, subdomain)) {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					handler := worker.NewWorker(appDir)
					handler.ServeHTTP(w, r)
				}),
			}

			if flags.tls {
				if flags.tlsCert == "" || flags.tlsKey == "" {
					return fmt.Errorf("TLS enabled, but no certificate or key provided")
				}

				cert, err := tls.LoadX509KeyPair(flags.tlsCert, flags.tlsKey)
				if err != nil {
					return fmt.Errorf("failed to load TLS certificate and key: %w", err)
				}

				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS12,
				}

				if flags.tlsCaCert != "" {
					caCert, err := os.ReadFile(flags.tlsCaCert)
					if err != nil {
						return fmt.Errorf("failed to read CA certificate: %w", err)
					}
					caCertPool := x509.NewCertPool()
					if !caCertPool.AppendCertsFromPEM(caCert) {
						return fmt.Errorf("failed to parse CA certificate")
					}
					tlsConfig.ClientCAs = caCertPool

					switch flags.tlsClientAuth {
					case "require":
						tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
					case "request":
						tlsConfig.ClientAuth = tls.RequestClientCert
					case "verify":
						tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
					default:
						return fmt.Errorf("invalid client auth option: %s", flags.tlsClientAuth)
					}
				}

				server.TLSConfig = tlsConfig

				cmd.Printf("Listening on https://%s\n", addr)
				return server.ListenAndServeTLS(flags.tlsCert, flags.tlsKey)
			}

			cmd.Printf("Listening on http://%s\n", addr)

			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			c := cron.New(cron.WithParser(parser))
			c.AddFunc("* * * * *", func() {
				fmt.Fprintln(os.Stderr, "Running cron jobs")
				entry := c.Entries()[0]

				apps, err := ListApps("")
				if err != nil {
					fmt.Println(err)
					return
				}

				for _, app := range apps {
					w := worker.NewWorker(app.Path)
					config, err := w.LoadConfig()
					if err != nil {
						continue
					}

					for _, job := range config.Crons {
						sched, err := parser.Parse(job.Schedule)
						if err != nil {
							fmt.Println(err)
							continue
						}

						if !sched.Next(entry.Prev).Equal(entry.Next) {
							continue
						}

						go func(job worker.Cron) error {
							if err := w.Run(job.Command, job.Args...); err != nil {
								fmt.Printf("Failed to run cron job %s: %s\n", job.Command, err)
							}

							return nil
						}(job)
					}

				}
			})

			go c.Start()

			return server.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&flags.host, "host", "localhost", "Host to listen on")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 0, "Port to listen on")
	cmd.Flags().BoolVar(&flags.tls, "tls", false, "Enable TLS")
	cmd.Flags().StringVar(&flags.tlsCert, "tls-cert", "", "TLS certificate file path")
	cmd.Flags().StringVar(&flags.tlsKey, "tls-key", "", "TLS key file path")
	cmd.Flags().StringVar(&flags.tlsCaCert, "tls-ca-cert", "", "TLS CA certificate file path")
	cmd.Flags().StringVar(&flags.tlsClientAuth, "tls-client-auth", "require", "TLS client auth mode (require, request, verify)")

	return cmd
}
