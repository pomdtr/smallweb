package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/pomdtr/smallweb/utils"
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
				return completeApp(cmd, args, toComplete)
			}

			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			apps, err := ListApps(k.String("domain"), rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %v", err)
			}

			var name string
			if args[0] == "." {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}

				if path.Dir(wd) != rootDir {
					return fmt.Errorf("not in a smallweb app directory")
				}

				name = path.Base(wd)
			} else {
				name = args[0]
			}

			for _, app := range apps {
				if app.Name != name {
					continue
				}

				worker, err := worker.NewWorker(app, k.StringMap("env"))
				if err != nil {
					return fmt.Errorf("could not create worker: %w", err)
				}

				cmd.SilenceErrors = true
				return worker.Run(args[1:])
			}

			return fmt.Errorf("app not found: %s", name)

		},
	}

	return cmd
}
