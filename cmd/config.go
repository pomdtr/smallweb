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
)

func findConfigPath(rootDir string) string {
	for _, candidate := range []string{".smallweb/config.jsonc", ".smallweb/config.json"} {
		configPath := filepath.Join(rootDir, candidate)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return filepath.Join(rootDir, ".smallweb/config.json")
}

func getEditorCmd(args ...string) (*exec.Cmd, error) {
	editorEnv, ok := os.LookupEnv("EDITOR")
	if !ok {
		return exec.Command("vi", args...), nil
	}

	editorArgs, err := shlex.Split(editorEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EDITOR: %w", err)
	}

	editorArgs = append(editorArgs, args...)
	return exec.Command(editorArgs[0], editorArgs[1:]...), nil
}

func NewCmdConfig() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:   "config <key>",
		Short: "Get a configuration value",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if flags.json {
					configBytes, err := k.Marshal(utils.ConfigParser())
					if err != nil {
						return err
					}

					os.Stdout.Write(configBytes)
					return nil
				}

				if !isatty.IsTerminal(os.Stdin.Fd()) {
					return fmt.Errorf("stdin is not interactive")
				}

				configPath := findConfigPath(k.String("dir"))
				editorCmd, err := getEditorCmd(configPath)
				if err != nil {
					return err
				}

				editorCmd.Stdin = os.Stdin
				editorCmd.Stdout = os.Stdout
				editorCmd.Stderr = os.Stderr

				return editorCmd.Run()
			}

			v := k.Get(args[0])
			if v == nil {
				return fmt.Errorf("key %q not found", args[0])
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(v); err != nil {
					return err
				}

				return nil
			}

			fmt.Println(v)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "Output as JSON")

	return cmd
}
