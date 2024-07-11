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
	"github.com/spf13/cobra"
)

func SplitHost(host string) (string, string, error) {
	// Special TLDs, as per RFC 2606
	parts := strings.Split(host, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", "", fmt.Errorf("invalid host")
	}

	tld := parts[len(parts)-1]
	if tld == "localhost" || tld == "test" || tld == "example" || tld == "invalid" {
		if len(parts) > 2 {
			return "", "", fmt.Errorf("invalid host")
		}

		return tld, parts[0], nil
	}

	if len(parts) == 2 {
		return parts[0] + "." + parts[1], "", nil
	}

	return parts[1] + "." + parts[2], parts[0], nil
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
					domain, subdomain, err := SplitHost(r.Host)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					if subdomain == "" {
						http.Redirect(w, r, fmt.Sprintf("https://www.%s", domain), http.StatusTemporaryRedirect)
						return
					}

					var handler http.Handler
					appDir := filepath.Join(worker.SMALLWEB_ROOT, domain, subdomain)
					if !worker.Exists(appDir) {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					handler = worker.NewWorker(appDir)
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
