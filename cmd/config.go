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

			if flags.json || !isatty.IsTerminal(os.Stdout.Fd()) {
				b, err := k.Marshal(utils.ConfigParser())
				if err != nil {
					return err
				}

				os.Stdout.Write(b)
				return nil
			}

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

			editorArgs, err := shlex.Split(findEditor())
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
