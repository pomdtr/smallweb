package cmd

import (
	"errors"
	"os/exec"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdGitReceivePack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-receive-pack",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.LoadApp(args[0], k.String("dir"), k.String("domain"))
			if err != nil {
				cmd.PrintErrf("failed to load app %s: %v\n", args[0], err)
				return ExitError{1}
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
			a, err := app.LoadApp(args[0], k.String("dir"), k.String("domain"))
			if err != nil {
				cmd.PrintErrf("failed to load app %s: %v\n", args[0], err)
				return ExitError{1}
			}

			gitCmd := exec.Command("git-upload-pack", a.BaseDir)

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
