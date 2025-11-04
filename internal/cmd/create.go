package cmd

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed templates/app/*
var appTemplateFS embed.FS

func NewCmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [app]",
		Args:  cobra.ExactArgs(1),
		Short: "Create a new Smallweb app",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(conf.String("dir"), args[0])

			if _, err := os.Stat(appDir); err == nil {
				cmd.PrintErrf("app %q already exists\n", args[0])
				return ExitError{1}
			}

			if err := os.MkdirAll(appDir, 0755); err != nil {
				return err
			}

			subFS, err := fs.Sub(appTemplateFS, "templates/app")
			if err != nil {
				cmd.PrintErrf("failed to create sub filesystem: %v\n", err)
				return ExitError{1}
			}

			if err := os.CopyFS(appDir, subFS); err != nil {
				cmd.PrintErrf("failed to extract app template: %v\n", err)
				return ExitError{1}
			}

			cmd.Printf("Created new app in %s\n", strings.Replace(appDir, os.Getenv("HOME"), "~", 1))
			return nil
		},
	}

	return cmd
}
