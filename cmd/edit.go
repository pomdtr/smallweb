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
		Use:     "edit <app> [file]",
		Short:   "Open an app in your editor",
		GroupID: CoreGroupID,
		Args:    cobra.RangeArgs(1, 2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			if len(args) == 0 {
				return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
			}

			// complete files in app directory
			if len(args) == 1 {
				appDir := filepath.Join(rootDir, args[0])
				if utils.FileExists(appDir) {
					entries, err := os.ReadDir(appDir)
					if err != nil {
						return nil, cobra.ShellCompDirectiveError
					}

					var files []string
					for _, entry := range entries {
						if !entry.IsDir() {
							files = append(files, entry.Name())
						}
					}

					return files, cobra.ShellCompDirectiveNoFileComp
				}
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			for _, appname := range ListApps(rootDir) {
				if appname == args[0] {
					appDir := filepath.Join(rootDir, appname)

					a, err := app.NewApp(appDir, fmt.Sprintf("https://%s.%s/", appname, k.String("domain")), k.StringMap("env"))
					if err != nil {
						return fmt.Errorf("failed to load app: %v", err)
					}

					var entrypoint string
					if len(args) == 2 {
						entrypoint = filepath.Join(a.Dir, args[1])
					} else {
						entrypoint = a.Entrypoint()

						if !utils.FileExists(entrypoint) {
							if utils.FileExists(filepath.Join(a.Root(), "index.html")) {
								entrypoint = filepath.Join(a.Root(), "index.html")
							} else {
								entries, err := os.ReadDir(a.Root())
								if err != nil {
									return fmt.Errorf("failed to read directory: %v", err)
								}

								if len(entries) == 0 {
									return fmt.Errorf("no files in app directory")
								}

								entrypoint = filepath.Join(a.Root(), entries[0].Name())
							}
						}
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
