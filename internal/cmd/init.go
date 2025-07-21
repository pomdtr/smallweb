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
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := k.String("dir")
			if dir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current working directory: %w", err)
				}

				dir = cwd
			}

			domain := k.String("domain")
			if domain == "" {
				domain = "smallweb.traefik.me"
			}

			subFS, err := fs.Sub(embedFS, "templates/workspace")
			if err != nil {
				return fmt.Errorf("failed to read workspace embed: %w", err)
			}

			templateFS := gosod.New(subFS)
			if err := templateFS.Extract(dir, map[string]any{
				"Domain": domain,
			}); err != nil {
				return fmt.Errorf("failed to extract workspace: %w", err)
			}

			return nil
		},
	}

	return cmd
}
