package cmd

import "github.com/spf13/cobra"

func NewCmdSSHEntrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh-entrypoint",
		Hidden: true,
	}

	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdGitReceivePack())
	cmd.AddCommand(NewCmdGitUploadPack())

	return cmd
}
