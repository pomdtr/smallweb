package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pomdtr/smallweb/client"
	"github.com/spf13/cobra"
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve [app]",
		Short: "Serve a smallweb app",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			apps, err := listApps()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			return apps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDenoIfRequired(); err != nil {
				return err
			}

			port, _ := cmd.Flags().GetInt("port")
			if len(args) == 1 {
				worker, err := client.NewWorker(args[0])
				if err != nil {
					return fmt.Errorf("failed to create client: %v", err)
				}

				server := http.Server{
					Addr:    fmt.Sprintf(":%d", port),
					Handler: worker,
				}

				fmt.Fprintln(os.Stderr, "Listening on", server.Addr)
				return server.ListenAndServe()
			}

			server := http.Server{
				Addr: fmt.Sprintf(":%d", port),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					app := r.Header.Get("X-Smallweb-App")
					if app == "" {
						http.Error(w, "No app specified", http.StatusBadRequest)
						return
					}

					worker, err := client.NewWorker(app)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					worker.ServeHTTP(w, r)
				}),
			}

			fmt.Fprintln(os.Stderr, "Listening on", server.Addr)
			return server.ListenAndServe()
		},
	}
	cmd.Flags().IntP("port", "p", 8000, "Port to listen on")
	return cmd
}
