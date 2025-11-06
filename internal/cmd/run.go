package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/internal/api"
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
			var appConfig app.Config
			if err := conf.Unmarshal(fmt.Sprintf("apps.%s", args[0]), &appConfig); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "failed to get app config: %v\n", err)
				return ExitError{1}
			}

			a, err := app.LoadApp(filepath.Join(conf.String("dir"), args[0]), appConfig)
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			wk := worker.NewWorker(a, api.NewHandler(conf))

			cmd.SilenceErrors = true

			params := worker.RunParams{
				Args:   args[1:],
				Stdout: cmd.OutOrStdout(),
				Stderr: cmd.ErrOrStderr(),
			}

			if !isatty.IsTerminal(os.Stdin.Fd()) {
				params.Stdin = cmd.InOrStdin()
			}

			if err := wk.Run(cmd.Context(), params); err != nil {
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
