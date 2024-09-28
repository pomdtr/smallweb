package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		SilenceErrors:      true,
		ValidArgsFunction:  completeApp(utils.ExpandTilde(k.String("dir"))),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}

			rootDir := utils.ExpandTilde(k.String("dir"))
			app, err := app.LoadApp(filepath.Join(rootDir, args[0]), k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}

			if strings.HasPrefix(app.Config.Entrypoint, "smallweb:") {
				return fmt.Errorf("smallweb built-in apps cannot be run")
			}

			worker := worker.NewWorker(app, k.StringMap("env"), nil)
			command, err := worker.Command(args[1:]...)
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			return command.Run()
		},
	}

	return cmd
}
