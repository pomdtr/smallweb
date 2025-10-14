package cmd

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/spf13/cobra"
)

//go:embed templates/app/*
var appFS embed.FS

func NewCmdCreate() *cobra.Command {
	var flags struct {
		template string
	}

	cmd := &cobra.Command{
		Use:   "create [app]",
		Args:  cobra.ExactArgs(1),
		Short: "Create a new Smallweb app",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(k.String("dir"), args[0])
			if _, err := os.Stat(appDir); !os.IsNotExist(err) {
				cmd.PrintErrf("App directory %s already exists.\n", appDir)
				return ExitError{1}
			}

			configDir, err := os.UserConfigDir()
			if err != nil {
				cmd.PrintErrf("failed to get user config directory: %v\n", err)
				return ExitError{1}
			}

			templatesDir := filepath.Join(configDir, "templates")
			var templateFS fs.FS
			if flags.template != "" {
				templateDir := filepath.Join(templatesDir, flags.template)
				if _, err := os.Stat(templateDir); os.IsNotExist(err) {
					cmd.PrintErrf("Template %s not found in %s.\n", flags.template, templatesDir)
					return ExitError{1}
				}

				templateFS = os.DirFS(templateDir)
			} else if _, err := os.Stat(filepath.Join(templatesDir, "default")); err == nil {
				templateFS = os.DirFS(filepath.Join(templatesDir, "default"))
			} else {
				templateFS = appFS
			}

			if err := os.CopyFS(appDir, templateFS); err != nil {
				cmd.PrintErrf("failed to copy app template: %v\n", err)
				return ExitError{1}
			}

			cmd.Printf("Created new app in %s\n", appDir)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.template, "template", "t", "", "Template to use for the new app")
	cmd.RegisterFlagCompletionFunc("template", completeTemplate)

	return cmd
}

func completeTemplate(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
	_ = k.Load(flagProvider, nil)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, err := os.ReadDir(filepath.Join(configDir, "templates"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		completions = append(completions, entry.Name())
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
