package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve <app>",
		Short: "Serve a smallweb app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			worker, err := NewHandler(args[0])
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			server := http.Server{
				Addr:    fmt.Sprintf(":%d", port),
				Handler: worker,
			}

			fmt.Fprintln(os.Stderr, "Listening on", server.Addr)
			return server.ListenAndServe()
		},
	}
	cmd.Flags().IntP("port", "p", 8000, "Port to listen on")
	return cmd
}
