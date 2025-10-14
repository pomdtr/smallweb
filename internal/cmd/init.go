package cmd

import (
	"embed"
	"io/fs"
	"os"

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
			dir := k.String("dir")
			if dir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					cmd.PrintErrf("failed to get current working directory: %v\n", err)
					return ExitError{1}
				}

				dir = cwd
			}

			subFS, err := fs.Sub(workspaceFS, "templates/workspace")
			if err != nil {
				cmd.PrintErrf("failed to create sub filesystem: %v\n", err)
				return ExitError{1}
			}

			if err := os.CopyFS(dir, subFS); err != nil {
				cmd.PrintErrf("failed to copy template files: %v\n", err)
				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}
