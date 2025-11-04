package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/matthewmueller/jsonc"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func findConfigPath(root string) string {
	for _, filename := range []string{"config.json", "config.jsonc"} {
		path := filepath.Join(root, ".smallweb", filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return filepath.Join(root, ".smallweb", "config.json")
}

func NewCmdConfig() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Open Smallweb configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := findConfigPath(conf.String("dir"))
			if flags.json || !isatty.IsTerminal(os.Stdout.Fd()) {
				configBytes, err := os.ReadFile(configPath)
				if err != nil {
					cmd.PrintErrf("failed to read config file: %v\n", err)
					return ExitError{1}
				}

				jsonBytes, err := jsonc.Standardize(configBytes)
				if err != nil {
					cmd.PrintErrf("failed to standardize config file: %v\n", err)
					return ExitError{1}
				}

				var config map[string]any
				if err := json.Unmarshal(jsonBytes, &config); err != nil {
					cmd.PrintErrf("failed to unmarshal config file: %v\n", err)
					return ExitError{1}
				}

				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)

				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(config); err != nil {
					cmd.PrintErrf("failed to encode config file: %v\n", err)
					return ExitError{1}
				}

				return nil
			}

			editor := "vi"
			if editorEnv, ok := os.LookupEnv("EDITOR"); ok {
				editor = editorEnv
			}

			editCmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", editor, configPath))

			editCmd.Stdout = os.Stdout
			editCmd.Stderr = os.Stderr
			editCmd.Stdin = os.Stdin

			if err := editCmd.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return ExitError{exitErr.ExitCode()}
				}

				return ExitError{1}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "Output the configuration in JSON format")

	return cmd
}
