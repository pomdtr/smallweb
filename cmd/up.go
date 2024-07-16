package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/pomdtr/smallweb/worker"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
)

type GlobalConfig struct {
	Host    string            `json:"host"`
	Port    int               `json:"port"`
	Domains map[string]string `json:"domains"`
}

var globalConfig GlobalConfig = GlobalConfig{
	Host: "localhost",
	Port: 7777,
	Domains: map[string]string{
		"*.localhost": "~/localhost",
	},
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("could not determine config home: %v", err)
	}
	var configHome = os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	configPath := filepath.Join(configHome, "smallweb", "config.json")
	if worker.Exists(configPath) {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatalf("could not read config.json: %v", err)
		}

		if err := json.Unmarshal(configBytes, &globalConfig); err != nil {
			log.Fatalf("could not unmarshal config.json: %v", err)
		}

		for domain, rootDir := range globalConfig.Domains {
			if strings.HasPrefix(rootDir, "~/") {
				globalConfig.Domains[domain] = filepath.Join(homeDir, rootDir[2:])
			}
		}
	}
}

func IsWildcard(domain string) bool {
	return strings.HasPrefix(domain, "*.")
}

func WorkerFromHostname(hostname string) (*worker.Worker, error) {
	if rootDir, ok := globalConfig.Domains[hostname]; ok {
		return &worker.Worker{Dir: rootDir}, nil
	}

	for domain, rootDir := range globalConfig.Domains {
		g, err := glob.Compile(domain)
		if err != nil {
			return nil, err
		}

		if !g.Match(hostname) {
			continue
		}

		if !IsWildcard(domain) {
			return &worker.Worker{Dir: rootDir}, nil
		}

		subdomain := strings.Split(hostname, ".")[0]
		return &worker.Worker{Dir: filepath.Join(rootDir, subdomain)}, nil
	}

	return nil, fmt.Errorf("domain not found")
}

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
					handler, err := WorkerFromHostname(r.Host)
					if err != nil {
						http.Error(w, "Not found", http.StatusNotFound)
					}

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

				apps, err := ListApps()
				if err != nil {
					fmt.Println(err)
					return
				}

				for _, app := range apps {
					w := worker.Worker{Dir: app.Dir}
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

						if sched.Next(rounded.Add(-1*time.Second)) != rounded {
							continue
						}

						go func(job worker.CronJob) error {
							if err := w.Trigger(job.Name); err != nil {
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
