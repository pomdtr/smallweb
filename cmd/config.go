package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/shlex"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [key]",
		Short: "Open the smallweb config in your editor",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := filepath.Join(utils.RootDir(), ".smallweb", "config.json")
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

			if len(args) > 0 {
				if args[0] == "dir" {
					fmt.Println(utils.RootDir())
					return nil
				}

				v := k.Get(args[0])
				if v == nil {
					return fmt.Errorf("key %q not found", args[0])
				}

				fmt.Println(v)
				return nil
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

	return cmd
}
