package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdEdit() *cobra.Command {
	var flags struct {
		file string
	}

	cmd := &cobra.Command{
		Use:               "edit <app>",
		Short:             "Open an app in your editor",
		Args:              cobra.ExactArgs(1),
		GroupID:           CoreGroupID,
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))

			app, err := app.LoadApp(filepath.Join(rootDir, args[0]))
			if err != nil {
				return fmt.Errorf("failed to get app: %v", err)
			}

			var file string
			if flags.file != "" {
				file = filepath.Join(app.Dir, flags.file)
			} else {
				entrypoint := app.Config.Entrypoint
				if strings.HasPrefix(entrypoint, "jsr:") || strings.HasPrefix(entrypoint, "npm:") || strings.HasPrefix(entrypoint, "smallweb:") {

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

	cmd.Flags().StringVarP(&flags.file, "file", "f", "", "File to edit")
	return cmd
}
