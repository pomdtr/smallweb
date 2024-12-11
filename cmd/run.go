package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run <app> [args...]",
		Short:             "Run an app cli",
		ValidArgsFunction: completeApp(k.String("dir")),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}

			a, err := app.NewApp(args[0], k.String("dir"), k.String("domain"), slices.Contains(k.Strings("adminApps"), args[0]))
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			wk := worker.NewWorker(a, k.String("dir"), k.String("domain"))
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
