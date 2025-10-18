package cmd

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdGit() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git",
		Hidden: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	cmd.AddCommand(NewCmdGitReceivePack())
	cmd.AddCommand(NewCmdGitUploadPack())

	return cmd
}

func NewCmdGitReceivePack() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-receive-pack",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				cmd.PrintErrf("invalid repository name %q\n", args[0])
				return ExitError{1}
			}

			a, err := app.LoadApp(k.String("dir"), parts[0], parts[1])
			if err != nil {
				cmd.PrintErrf("failed to load app %s: %v\n", args[0], err)
				return ExitError{1}
			}

			gitCmd := exec.Command("git-receive-pack", a.Dir)

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
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				cmd.PrintErrf("invalid repository name %q\n", args[0])
				return ExitError{1}
			}

			a, err := app.LoadApp(k.String("dir"), parts[0], parts[1])
			if err != nil {
				cmd.PrintErrf("failed to load app %s: %v\n", args[0], err)
				return ExitError{1}
			}

			gitCmd := exec.Command("git-upload-pack", a.Dir)

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
