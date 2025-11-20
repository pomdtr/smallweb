package cmd

import (
	"errors"
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
			a, err := app.LoadApp(filepath.Join(conf.String("dir"), args[0]))
			if err != nil {
				cmd.PrintErrf("failed to load app %s: %v\n", args[0], err)
				return ExitError{1}
			}

			gitCmd := exec.Command("git", "receive-pack", a.Dir)

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
			a, err := app.LoadApp(filepath.Join(conf.String("dir"), args[0]))
			if err != nil {
				cmd.PrintErrf("failed to load app %s: %v\n", args[0], err)
				return ExitError{1}
			}

			gitCmd := exec.Command("git", "upload-pack", a.Dir)

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
