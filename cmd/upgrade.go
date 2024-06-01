package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmdUpgrade() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade smallweb",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("TODO")
			return nil
		},
	}
}
