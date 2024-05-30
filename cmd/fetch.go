package cmd

import "github.com/spf13/cobra"

func NewCmdFetch() *cobra.Command {
	var flags struct {
		Method string
		Header []string
	}

	cmd := &cobra.Command{
		Use: "fetch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Method, "method", "X", "GET", "HTTP method")
	cmd.Flags().StringSliceVarP(&flags.Header, "header", "H", nil, "HTTP header")

	return cmd
}
