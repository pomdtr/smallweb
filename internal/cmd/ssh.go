package cmd

import (
	"github.com/knadh/koanf/providers/posflag"
	"github.com/spf13/cobra"
)

func NewCmdSSH() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh",
		Hidden: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			flagProvider := posflag.Provider(cmd.Root().Flags(), ".", k)
			_ = k.Load(flagProvider, nil)
			return nil
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdShell())
	cmd.AddCommand(NewCmdGitReceivePack())
	cmd.AddCommand(NewCmdGitUploadPack())
	cmd.AddCommand(NewCmdCreate())

	cmd.PersistentFlags().String("dir", "", "The root directory for smallweb")
	cmd.PersistentFlags().String("domain", "", "")

	cmd.MarkFlagRequired("dir")
	cmd.MarkFlagRequired("domain")

	return cmd
}
