package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	return &cobra.Command{
		Use:                "run <alias> [args...]",
		Short:              "Run a smallweb app cli",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			apps, err := listApps()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			return apps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDenoIfRequired(); err != nil {
				return err
			}

			worker, err := NewHandler(args[0])
			if err != nil {
				return err
			}

			return worker.Run(args[1:])
		},
	}

}
