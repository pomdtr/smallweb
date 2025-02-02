package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
)

//go:embed embed/workspace/*
var embedFS embed.FS

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceFS, err := fs.Sub(embedFS, "embed/workspace")
			if err != nil {
				return fmt.Errorf("failed to read workspace embed: %w", err)
			}

			workspaceDir := k.String("dir")
			if _, err := fs.Stat(workspaceFS, workspaceDir); err == nil {
				return fmt.Errorf("directory %s already exists", workspaceDir)
			}

			if err := os.CopyFS(workspaceDir, workspaceFS); err != nil {
				return fmt.Errorf("failed to copy workspace embed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Workspace initialized at %s\n", workspaceDir)
			return nil
		},
	}

	return cmd
}
