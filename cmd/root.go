package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smallweb",
		Short: "Host websites from your internet folder",
	}

	cmd.AddCommand(NewCmdUp())
	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdService())
	cmd.AddCommand(NewCmdDump())
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdCreate())

	return cmd
}
