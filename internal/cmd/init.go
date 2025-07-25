package cmd

import (
	"embed"
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

			subFS, err := fs.Sub(embedFS, "templates/workspace")
			if err != nil {
				cmd.PrintErrf("failed to create sub filesystem: %v\n", err)
				return ExitError{1}
			}

			templateFS := gosod.New(subFS)
			if err := templateFS.Extract(dir, map[string]any{
				"Domain": domain,
			}); err != nil {
				cmd.PrintErrf("failed to extract workspace: %v\n", err)
				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}
