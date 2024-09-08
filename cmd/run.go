package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "run <app> [args...]",
		Short:              "Run an app cli",
		GroupID:            CoreGroupID,
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			if len(args) == 0 {
				return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}

			rootDir := utils.ExpandTilde(k.String("dir"))
			for _, appname := range ListApps(rootDir) {
				if appname != args[0] {
					continue
				}

				worker, err := app.NewApp(filepath.Join(rootDir, args[0]), fmt.Sprintf("https://%s.%s/", appname, k.String("domain")), k.StringMap("env"))
				if err != nil {
					return fmt.Errorf("could not create worker: %w", err)
				}

				cmd.SilenceErrors = true
				return worker.Run(args[1:]...)
			}

			return fmt.Errorf("app not found: %s", args[0])

		},
	}

	return cmd
}
