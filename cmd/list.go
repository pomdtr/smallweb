package cmd

import "github.com/spf13/cobra"

func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all apps",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
}
