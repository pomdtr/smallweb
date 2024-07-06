package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdUp() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Start the smallweb evaluation server",
		GroupID: CoreGroupID,
		Aliases: []string{"serve"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			server := http.Server{
				Addr: fmt.Sprintf(":%d", port),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handler, err := worker.WorkerFromHost(r.Host)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
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
