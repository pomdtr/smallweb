package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "run <app> [args...]",
		Short:              "Run an app cli",
		GroupID:            CoreGroupID,
		DisableFlagParsing: true,
		ValidArgsFunction:  completeApp(utils.ExpandTilde(k.String("dir"))),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}

			rootDir := utils.ExpandTilde(k.String("dir"))
			app, err := app.LoadApp(filepath.Join(rootDir, args[0]))
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}

			worker := worker.NewWorker(app, k.StringMap("env"))
			return worker.Run(args[1:]...)
		},
	}

	return cmd
}
