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

func NewCmdEdit() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "edit <app>",
		Short:             "Open an app in your editor",
		Args:              cobra.ExactArgs(1),
		GroupID:           CoreGroupID,
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))

			if (len(args)) == 0 {
				return fmt.Errorf("app name is required")
			}

			app, err := app.LoadApp(filepath.Join(rootDir, args[0]), k.String("domains.base"))
			if err != nil {
				return fmt.Errorf("failed to get app: %v", err)
			}

			var file string
			if utils.FileExists(app.Entrypoint()) {
				file = app.Entrypoint()
			} else if utils.FileExists(filepath.Join(app.Root(), "index.html")) {
				file = filepath.Join(app.Root(), "index.html")
			} else if utils.FileExists(filepath.Join(app.Root(), "smallweb.json")) {
				file = filepath.Join(app.Root(), "smallweb.json")
			} else {
				return fmt.Errorf("no entrypoint found")
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

	return cmd
}
