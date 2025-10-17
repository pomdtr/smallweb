package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdSSH() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh",
		Hidden: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdShell())
	cmd.AddCommand(NewCmdGitReceivePack())
	cmd.AddCommand(NewCmdGitUploadPack())

	return cmd
}
