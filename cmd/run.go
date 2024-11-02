package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
			app, err := app.LoadApp(filepath.Join(rootDir, args[0]), k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}

			if strings.HasPrefix(app.Config.Entrypoint, "smallweb:") {
				return fmt.Errorf("smallweb built-in apps do not support running as a CLI")
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
