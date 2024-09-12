package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/shlex"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func findConfigPath() string {
	if config, ok := os.LookupEnv("SMALLWEB_CONFIG"); ok {
		return config
	}

	var configDir string
	if configHome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		configDir = filepath.Join(configHome, "smallweb")
	} else {
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "smallweb")
	}

	if utils.FileExists(filepath.Join(configDir, "config.jsonc")) {
		return filepath.Join(configDir, "config.jsonc")
	}

	return filepath.Join(configDir, "config.json")
}

func NewCmdEdit() *cobra.Command {
	var flags struct {
		file   string
		config bool
	}

	cmd := &cobra.Command{
		Use:               "edit <app>",
		Short:             "Open an app in your editor",
		Args:              cobra.MaximumNArgs(1),
		GroupID:           CoreGroupID,
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))

			if flags.config {
				configPath := findConfigPath()
				editorCmd := k.String("editor")
				editorArgs, err := shlex.Split(editorCmd)
				if err != nil {
					return err
				}

				editorArgs = append(editorArgs, configPath)
				command := exec.Command(editorArgs[0], editorArgs[1:]...)
				command.Stdin = os.Stdin
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				command.Dir = filepath.Dir(configPath)

				if err := command.Run(); err != nil {
					return fmt.Errorf("failed to run editor: %v", err)
				}

				return nil
			}

			app, err := app.LoadApp(filepath.Join(rootDir, args[0]))
			if err != nil {
				return fmt.Errorf("failed to get app: %v", err)
			}

			file := flags.file
			if file == "" {
				if utils.FileExists(app.Entrypoint()) {
					file = app.Entrypoint()
				} else if utils.FileExists(filepath.Join(app.Root(), "index.html")) {
					file = filepath.Join(app.Root(), "index.html")
				} else if utils.FileExists(filepath.Join(app.Root(), "smallweb.json")) {
					file = filepath.Join(app.Root(), "smallweb.json")
				} else {
					return NewExitError(1, "could not find a file to edit, please specify one with --file")
				}
			}

			editorCmd := k.String("editor")
			editorArgs, err := shlex.Split(editorCmd)
			if err != nil {
				return err
			}

			editorArgs = append(editorArgs, file)
			command := exec.Command(editorArgs[0], editorArgs[1:]...)
			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			command.Dir = app.Dir

			if err := command.Run(); err != nil {
				return fmt.Errorf("failed to run editor: %v", err)
			}

			return nil

		},
	}

	cmd.Flags().BoolVarP(&flags.config, "config", "c", false, "Edit config file")
	cmd.Flags().StringVarP(&flags.file, "file", "f", "", "File to edit")
	cmd.MarkFlagsMutuallyExclusive("file", "config")
	return cmd
}
