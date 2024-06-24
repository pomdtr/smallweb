package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use: "smallweb",
	}

	cmd.AddCommand(NewCmdUp())
	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdService())

	return cmd
}
