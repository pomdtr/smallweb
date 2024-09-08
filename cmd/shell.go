package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/abiosoft/ishell/v2"
	"github.com/abiosoft/readline"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdShell() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Start a shell",
		Run: func(cmd *cobra.Command, args []string) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			shell := ishell.NewWithConfig(&readline.Config{
				Prompt: "$ ",
			})

			for _, appname := range ListApps(rootDir) {
				shell.AddCmd(&ishell.Cmd{
					Name: appname,
					Func: func(c *ishell.Context) {
						a, err := app.NewApp(filepath.Join(rootDir, appname), fmt.Sprintf("https://%s.%s/", appname, k.String("domain")), k.StringMap("env"))
						if err != nil {
							c.Println(err)
							return
						}

						output, err := a.Output(c.Args...)
						if err != nil {
							var exitErr *exec.ExitError
							if errors.As(err, &exitErr) {
								c.Print(string(exitErr.Stderr))
								return
							}

							c.Print(err.Error())
							return
						}

						c.Print(string(output))
					},
				})
			}

			shell.Run()
		},
	}

	return cmd
}
