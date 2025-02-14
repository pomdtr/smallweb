package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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

			domain := k.String("domain")
			if domain == "" {
				return fmt.Errorf("domain is required")
			}

			if err := os.CopyFS(workspaceDir, workspaceFS); err != nil {
				return fmt.Errorf("failed to copy workspace embed: %w", err)
			}

			if err := os.Mkdir(filepath.Join(workspaceDir, ".smallweb"), 0755); err != nil {
				return fmt.Errorf("failed to create .smallweb directory: %w", err)
			}

			configBytes, err := json.MarshalIndent(map[string]interface{}{
				"domain": domain,
			}, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}
			configBytes = append(configBytes, '\n')

			if err := os.WriteFile(filepath.Join(workspaceDir, ".smallweb", "config.json"), configBytes, 0644); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Workspace initialized at %s\n", workspaceDir)
			return nil
		},
	}

	return cmd
}
