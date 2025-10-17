package cmd

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/worker"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run <app> [args...]",
		Short:             "Run an app cli",
		ValidArgsFunction: completeApp,
		Args:              cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := checkDenoVersion(); err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.LoadApp(k.String("dir"), k.String("domain"), args[0])
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			wk := worker.NewWorker(a, nil)
			command, err := wk.Command(cmd.Context(), args[1:])
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			cmd.SilenceErrors = true

			command.Stdin = cmd.InOrStdin()
			command.Stdout = cmd.OutOrStdout()
			command.Stderr = cmd.ErrOrStderr()
			if err := command.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return ExitError{exitErr.ExitCode()}
				}

				return ExitError{1}
			}

			return nil

		},
	}

	cmd.Flags().SetInterspersed(false)
	return cmd
}
