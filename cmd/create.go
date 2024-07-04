package cmd

import (
	"path"

	"github.com/pomdtr/smallweb/templates"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdCreate() *cobra.Command {
	var flags struct {
		template string
	}

	cmd := &cobra.Command{
		Use:   "create <app>",
		Short: "Create a new smallweb app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dst := path.Join(worker.SMALLWEB_ROOT, args[0])
			return templates.Install(flags.template, dst)
		},
	}

	cmd.Flags().StringVarP(&flags.template, "template", "t", "", "The template to use")
	cmd.RegisterFlagCompletionFunc("template", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		options, err := templates.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return options, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.MarkFlagRequired("template")

	return cmd
}
