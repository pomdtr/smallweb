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
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "edit <app>",
		Short:   "Open an app in your editor",
		GroupID: CoreGroupID,
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			for _, appname := range ListApps(rootDir) {
				if appname == args[0] {
					appDir := filepath.Join(rootDir, appname)

					a, err := app.NewApp(appDir, fmt.Sprintf("%s.%s", appname, k.String("domain")), k.StringMap("env"))
					if err != nil {
						return fmt.Errorf("failed to load app: %v", err)
					}

					entrypoint := a.Entrypoint()
					if !utils.FileExists(entrypoint) {
						return fmt.Errorf("entrypoint is not a file: %s", entrypoint)
					}

					editorCmd := k.String("editor")
					editorArgs, err := shlex.Split(editorCmd)
					if err != nil {
						return err
					}

					editorArgs = append(editorArgs, entrypoint)
					command := exec.Command(editorArgs[0], editorArgs[1:]...)
					command.Stdin = os.Stdin
					command.Stdout = os.Stdout
					command.Stderr = os.Stderr
					command.Dir = appDir

					if err := command.Run(); err != nil {
						return fmt.Errorf("failed to run editor: %v", err)
					}

					return nil

				}
			}

			return fmt.Errorf("app %s not found", args[0])
		},
	}

	cmd.Flags().BoolVarP(&flags.json, "json", "j", false, "Output as JSON")
	return cmd
}
