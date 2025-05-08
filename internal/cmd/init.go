package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/leaanthony/gosod"
	"github.com/spf13/cobra"
)

//go:embed templates/workspace/*
var embedFS embed.FS

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := k.String("dir")
			if dir == "" {
				return fmt.Errorf("dir is required")
			}

			domain := k.String("domain")
			if domain == "" {
				return fmt.Errorf("domain is required")
			}

			subFS, err := fs.Sub(embedFS, "templates/workspace")
			if err != nil {
				return fmt.Errorf("failed to read workspace embed: %w", err)
			}

			if _, err := os.Stat(dir); err == nil {
				entries, err := os.ReadDir(dir)
				if err != nil {
					return fmt.Errorf("failed to read directory %s: %w", dir, err)
				}

				if len(entries) > 0 {
					return fmt.Errorf("directory %s already exists and is not empty", dir)
				}
			}

			templateFS := gosod.New(subFS)
			if err := templateFS.Extract(dir, map[string]any{
				"Domain": domain,
			}); err != nil {
				return fmt.Errorf("failed to extract workspace: %w", err)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Workspace initialized at %s\n", dir)
			return nil
		},
	}

	return cmd
}
