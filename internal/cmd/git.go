package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdGitReceivePack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-receive-pack",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.LoadApp(args[0], k.String("dir"))
			if err != nil {

			}

			gitCmd := exec.Command("git-receive-pack", a.BaseDir)

			gitCmd.Stdin = cmd.InOrStdin()
			gitCmd.Stdout = cmd.OutOrStdout()
			gitCmd.Stderr = cmd.ErrOrStderr()

			if err := gitCmd.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return ExitError{exitErr.ExitCode()}
				}

				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}

func NewCmdGitUploadPack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-upload-pack",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(k.String("dir"), args[0])
			if _, err := os.Stat(appDir); os.IsNotExist(err) {
				cmd.PrintErrln("app not found:", args[0])
				return ExitError{1}
			}

			gitCmd := exec.Command("git-upload-pack", appDir)

			gitCmd.Stdin = cmd.InOrStdin()
			gitCmd.Stdout = cmd.OutOrStdout()
			gitCmd.Stderr = cmd.ErrOrStderr()

			if err := gitCmd.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return ExitError{exitErr.ExitCode()}
				}

				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}
