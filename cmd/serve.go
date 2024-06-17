package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pomdtr/smallweb/client"
	"github.com/spf13/cobra"
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve a smallweb app",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDenoIfRequired(); err != nil {
				return err
			}

			port, _ := cmd.Flags().GetInt("port")
			server := http.Server{
				Addr: fmt.Sprintf(":%d", port),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					app := r.Header.Get("X-Smallweb-App")

					// if app is not provided, infer it from the subdomain
					if app == "" {
						parts := strings.Split(r.Host, ".")
						app = parts[0]
					}

					handler, err := client.NewWorker(app)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					handler.ServeHTTP(w, r)
				}),
			}

			fmt.Fprintln(os.Stderr, "Listening on", server.Addr)
			return server.ListenAndServe()
		},
	}
	cmd.Flags().IntP("port", "p", 8000, "Port to listen on")
	return cmd
}
