package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/v2"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/config"
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
			a, err := app.LoadApp(filepath.Join(k.String("dir"), args[0]))
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			var permissionSet config.PermissionSet
			if key := fmt.Sprintf("permissions.%s", a.Name); k.Exists(key) {
				if err := k.UnmarshalWithConf(key, &permissionSet, koanf.UnmarshalConf{
					DecoderConfig: &mapstructure.DecoderConfig{
						DecodeHook: config.PermissionConfigDecodeHook(),
					},
				}); err != nil {
					cmd.PrintErrf("failed to parse permissions from config: %v\n", err)
				}
			}

			wk := worker.NewWorker(a, &permissionSet)
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
