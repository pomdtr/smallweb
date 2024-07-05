package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the smallweb evaluation server",
		Aliases: []string{"up"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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

					handler, err := worker.NewWorker(app)
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
	cmd.Flags().IntP("port", "p", 7777, "Port to listen on")
	return cmd
}
