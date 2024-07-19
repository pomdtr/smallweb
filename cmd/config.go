package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/shlex"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func findConfigPath() string {
	if env, ok := os.LookupEnv("SMALLWEB_CONFIG"); ok {
		return env
	}

	var configDir string
	if env, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		configDir = filepath.Join(env, "smallweb", "config.json")
	} else {
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "smallweb")
	}

	if utils.FileExists(filepath.Join(configDir, "config.jsonc")) {
		return filepath.Join(configDir, "config.jsonc")
	}

	return filepath.Join(configDir, "config.json")
}

func findEditor() string {
	if env, ok := os.LookupEnv("EDITOR"); ok {
		return env
	}

	return "vim"
}

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Open the smallweb config in your editor",
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			editorCmd := findEditor()
			editorArgs, err := shlex.Split(editorCmd)
			if err != nil {
				return err
			}
			editorArgs = append(editorArgs, findConfigPath())

			command := exec.Command(editorArgs[0], editorArgs[1:]...)
			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if err := command.Run(); err != nil {
				return fmt.Errorf("failed to run editor: %v", err)
			}

			return nil
		},
	}

	return cmd
}
