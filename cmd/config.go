package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/shlex"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"github.com/tailscale/hujson"
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

func findEditor() string {
	if env, ok := os.LookupEnv("EDITOR"); ok {
		return env
	}

	return "vim"
}

func NewCmdConfig() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Open the smallweb config in your editor",
		GroupID: CoreGroupID,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := findConfigPath()
			if !utils.FileExists(configPath) {
				var config map[string]any
				if err := k.Unmarshal("", &config); err != nil {
					return err
				}

				if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
					return err
				}

				f, err := os.Create(configPath)
				if err != nil {
					return err
				}
				defer f.Close()

				encoder := json.NewEncoder(f)
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "  ")

				if err := encoder.Encode(config); err != nil {
					return err
				}
			}

			if flags.json || !isatty.IsTerminal(os.Stdout.Fd()) {
				b, err := os.ReadFile(configPath)
				if err != nil {
					return err
				}

				configBytes, err := hujson.Standardize(b)
				if err != nil {
					return err
				}

				var config map[string]any
				if err := json.Unmarshal(configBytes, &config); err != nil {
					return err
				}

				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(config); err != nil {
					return err
				}

				return nil
			}

			editorArgs, err := shlex.Split(k.String("editor"))
			if err != nil {
				return err
			}

			editorArgs = append(editorArgs, configPath)

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

	cmd.Flags().BoolVarP(&flags.json, "json", "j", false, "Output as JSON")
	return cmd
}
