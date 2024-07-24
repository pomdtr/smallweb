package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

func DirFromHostname(domains map[string]string, hostname string) (string, error) {
	if rootDir, ok := domains[hostname]; ok {
		return rootDir, nil
	}

	items := make([][]string, 0)
	for domain, rootDir := range domains {
		items = append(items, []string{domain, rootDir})
	}

	slices.SortFunc(items, func(a, b []string) int {
		return len(b[0]) - len(a[0])
	})

	for _, item := range items {
		domain, rootDir := item[0], item[1]
		match, err := utils.ExtractGlobPattern(hostname, domain)
		if err != nil {
			continue
		}

		return strings.Replace(rootDir, "*", match, 1), nil

	}

	return "", fmt.Errorf("domain not found")
}

func NewCmdUp() *cobra.Command {
	var flags struct {
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
			port := k.Int("port")
			if port == 0 && flags.tls {
				port = 443
			} else if port == 0 {
				port = 7777
			}

			host := k.String("host")
			addr := fmt.Sprintf("%s:%d", host, port)

			server := http.Server{
				Addr: addr,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					domains := expandDomains(k.StringMap("domains"))
					dir, err := DirFromHostname(domains, r.Host)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					handler, err := worker.NewWorker(dir, k.StringMap("env"))
					for k, v := range k.StringMap("env") {
						handler.Env[k] = v
					}

					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					handler.Env = k.StringMap("env")
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
				rounded := time.Now().Truncate(time.Minute)

				apps, err := ListApps(expandDomains(k.StringMap("domains")))
				if err != nil {
					fmt.Println(err)
					return
				}

				for _, app := range apps {
					w, err := worker.NewWorker(app.Dir, k.StringMap("env"))
					if err != nil {
						fmt.Println(err)
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

						go func(cron worker.CronJob) error {
							req, err := http.NewRequest("GET", cron.Path, nil)
							if err != nil {
								return err
							}
							req.Host = app.Name

							if secret, ok := w.Env["CRON_SECRET"]; ok {
								req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", secret))
							}

							resp, err := w.Do(req)
							if err != nil {
								return err
							}
							defer resp.Body.Close()

							if resp.StatusCode != http.StatusOK {
								return fmt.Errorf("non-200 status code: %d", resp.StatusCode)
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

	cmd.Flags().String("host", "localhost", "Host to listen on")
	cmd.Flags().IntP("port", "p", 0, "Port to listen on")
	cmd.Flags().BoolVar(&flags.tls, "tls", false, "Enable TLS")
	cmd.Flags().StringVar(&flags.tlsCert, "tls-cert", "", "TLS certificate file path")
	cmd.Flags().StringVar(&flags.tlsKey, "tls-key", "", "TLS key file path")
	cmd.Flags().StringVar(&flags.tlsCaCert, "tls-ca-cert", "", "TLS CA certificate file path")
	cmd.Flags().StringVar(&flags.tlsClientAuth, "tls-client-auth", "require", "TLS client auth mode (require, request, verify)")

	if err := k.Load(posflag.Provider(cmd.Flags(), ".", k), nil); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	return cmd
}
