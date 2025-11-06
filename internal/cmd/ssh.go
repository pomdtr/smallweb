package cmd

import "github.com/spf13/cobra"

func NewCmdSSHEntrypoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh-entrypoint",
		Hidden: true,
	}

	cmd.AddCommand(NewCmdCreate())
	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdRename())
	cmd.AddCommand(NewCmdDelete())
	cmd.AddCommand(NewCmdApi())
	cmd.AddCommand(NewCmdRun())

	cmd.AddCommand(NewCmdGitReceivePack())
	cmd.AddCommand(NewCmdGitUploadPack())

	return cmd
}
