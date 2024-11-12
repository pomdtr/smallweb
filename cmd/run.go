package cmd

import (
	"fmt"
	"os"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/config"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "run <app> [args...]",
		Short:              "Run an app cli",
		DisableFlagParsing: true,
		ValidArgsFunction:  completeApp(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}

			rootDir := utils.RootDir()
			app, err := app.LoadApp(args[0], config.Config{
				Dir:    rootDir,
				Domain: k.String("domain"),
			})
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}

			wk := worker.NewWorker(app, config.Config{
				Domain: k.String("domain"),
			})
			command, err := wk.Command(args[1:]...)
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			cmd.SilenceErrors = true

			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			return command.Run()
		},
	}

	return cmd
}
