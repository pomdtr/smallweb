//go:build windows

package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdService() *cobra.Command {
	cmd := &cobra.Command{
		Use: "service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("service command not supported on Windows yet")
			return nil
		},
	}

	return cmd
}
