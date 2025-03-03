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
			dir, _ := cmd.Flags().GetString("dir")
			if dir == "" {
				return fmt.Errorf("the dir flag is required")
			}

			domain, _ := cmd.Flags().GetString("domain")
			if domain == "" {
				return fmt.Errorf("the domain flag is required")
			}

			subFS, err := fs.Sub(embedFS, "templates/workspace")
			if err != nil {
				return fmt.Errorf("failed to read workspace embed: %w", err)
			}

			if _, err := os.Stat(k.String("dir")); err == nil {
				return fmt.Errorf("directory %s already exists", k.String("dir"))
			}

			templateFS := gosod.New(subFS)
			if err := templateFS.Extract(k.String("dir"), map[string]any{
				"Domain": k.String("domain"),
			}); err != nil {
				return fmt.Errorf("failed to extract workspace: %w", err)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Workspace initialized at %s\n", k.String("dir"))
			return nil
		},
	}

	return cmd
}
