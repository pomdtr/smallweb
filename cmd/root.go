package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdRoot(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "smallweb",
		Short:              "Host websites from your internet folder",
		DisableFlagParsing: true,
		Version:            version,
	}

	cmd.AddCommand(NewCmdServe())
	cmd.AddCommand(NewCmdService())
	cmd.AddCommand(NewCmdDump())
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdCreate())
	cmd.AddCommand(NewCmdUpgrade())

	return cmd
}
