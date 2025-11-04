package cmd

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

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
			socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("smallweb-%s.sock", rand.Text()))
			ln, err := net.Listen("unix", socketPath)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "failed to start api socket: %v\n", err)
				return ExitError{1}
			}

			api := api.NewHandler(conf)
			go http.Serve(ln, api)
			defer ln.Close()
			defer os.Remove(socketPath)

			var appConfig app.Config
			if err := conf.Unmarshal(fmt.Sprintf("apps.%s", args[0]), &appConfig); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "failed to get app config: %v\n", err)
				return ExitError{1}
			}

			a, err := app.LoadApp(filepath.Join(conf.String("dir"), args[0]), appConfig)
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			wk := worker.NewWorker(a, socketPath)
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
