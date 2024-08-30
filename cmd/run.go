package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "run <app> [args...]",
		Short:              "Run an app cli",
		GroupID:            CoreGroupID,
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return ListApps(), cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, app := range ListApps() {
				if app != args[0] {
					continue
				}

				worker, err := worker.NewWorker(filepath.Join(rootDir, args[0]))
				if err != nil {
					return fmt.Errorf("could not create worker: %w", err)
				}

				cmd.SilenceErrors = true
				return worker.Run(args[1:])
			}

			return fmt.Errorf("app not found: %s", args[0])

		},
	}

	return cmd
}
