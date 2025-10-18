package cmd

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed templates/workspace/*
var workspaceFS embed.FS

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Args:  cobra.NoArgs,
		Short: "Initialize a new workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := k.String("domain")
			if domain == "" {
				cmd.PrintErrf("--domain flag is required for init command")
				return ExitError{1}
			}

			dir := k.String("dir")
			if dir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					cmd.PrintErrf("failed to get current working directory: %v\n", err)
					return ExitError{1}
				}

				dir = cwd
			}

			workspaceDir := filepath.Join(dir, domain)
			subFS, err := fs.Sub(workspaceFS, "templates/workspace")
			if err != nil {
				cmd.PrintErrf("failed to create sub filesystem: %v\n", err)
				return ExitError{1}
			}

			if err := os.MkdirAll(workspaceDir, 0755); err != nil {
				cmd.PrintErrf("failed to create workspace directory %s: %v\n", workspaceDir, err)
				return ExitError{1}
			}

			if err := os.CopyFS(workspaceDir, subFS); err != nil {
				cmd.PrintErrf("failed to copy workspace template: %v\n", err)
				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}
