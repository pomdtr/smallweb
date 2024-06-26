package cmd

import (
	"os"
	"path/filepath"

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	return &cobra.Command{
		Use:                "run <alias> [args...]",
		Short:              "Run a smallweb app cli",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			entries, err := os.ReadDir(worker.SMALLWEB_ROOT)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var apps []string
			for _, entry := range entries {
				for _, extension := range worker.EXTENSIONS {
					if worker.FileExists(filepath.Join(worker.SMALLWEB_ROOT, entry.Name(), "cli"+extension)) {
						apps = append(apps, entry.Name())
						break
					}
				}
			}

			return apps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, err := worker.NewWorker(args[0])
			if err != nil {
				return err
			}

			return worker.Run(args[1:])
		},
	}

}
