package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"

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

func NewCmdWebdav() *cobra.Command {
	var flags struct {
		addr string
		user string
		pass string
		cert string
		key  string
	}

	cmd := &cobra.Command{
		Use:   "webdav",
		Short: "Start a webdav server",
		RunE: func(cmd *cobra.Command, args []string) error {
			var handler http.Handler = &webdav.Handler{
				FileSystem: webdav.Dir("."),
				LockSystem: webdav.NewMemLS(),
			}

			if flags.user != "" || flags.pass != "" {
				handler = basicAuth(handler, flags.user, flags.pass)
			}

			server := &http.Server{
				Addr:    flags.addr,
				Handler: handler,
			}

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

				cmd.Println("Serving webdav on", flags.addr)
				return server.ListenAndServeTLS(flags.cert, flags.key)
			}

			cmd.Println("Serving webdav on", flags.addr)
			return server.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&flags.addr, "addr", "localhost:8080", "address to listen on")
	cmd.Flags().StringVar(&flags.user, "user", "", "username")
	cmd.Flags().StringVar(&flags.pass, "pass", "", "password")
	cmd.Flags().StringVar(&flags.cert, "cert", "", "TLS certificate file")
	cmd.Flags().StringVar(&flags.key, "key", "", "TLS key file")
	return cmd
}
