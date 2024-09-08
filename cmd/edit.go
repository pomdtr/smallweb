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
		file string
	}

	cmd := &cobra.Command{
		Use:     "edit <app>",
		Short:   "Open an app in your editor",
		GroupID: CoreGroupID,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			if len(args) == 0 {
				return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			rootDir := utils.ExpandTilde(k.String("dir"))
			for _, appname := range ListApps(rootDir) {
				if appname == args[0] {
					appDir := filepath.Join(rootDir, appname)

					a, err := app.NewApp(appDir, fmt.Sprintf("https://%s.%s/", appname, k.String("domain")), k.StringMap("env"))
					if err != nil {
						return fmt.Errorf("failed to load app: %v", err)
					}

					var file string
					if flags.file != "" {
						file = filepath.Join(a.Dir, flags.file)
					} else {
						file = a.Entrypoint()
						if !utils.FileExists(file) {
							if utils.FileExists(filepath.Join(a.Root(), "index.html")) {
								file = filepath.Join(a.Root(), "index.html")
							} else {
								entries, err := os.ReadDir(a.Root())
								if err != nil {
									return fmt.Errorf("failed to read directory: %v", err)
								}

								if len(entries) == 0 {
									return fmt.Errorf("no files in app directory")
								}

								file = filepath.Join(a.Root(), entries[0].Name())
							}
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

	cmd.Flags().StringVarP(&flags.file, "file", "f", "", "File to edit")
	return cmd
}
