package cmd

import (
	"fmt"
	"os"
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
				Prompt: "\033[32m$\033[0m ",
			})

			shell.AddCmd(&ishell.Cmd{
				Name: fmt.Sprintf("cli.%s", k.String("domain")),
				Func: func(c *ishell.Context) {
					execPath, err := os.Executable()
					if err != nil {
						c.Println(err)
						c.Err(err)
						return
					}

					command := exec.Command(execPath, c.Args...)
					output, err := command.CombinedOutput()
					if err != nil {
						c.Err(err)
					}

					c.Print(string(output))
				},
			})

			for _, appname := range ListApps(rootDir) {
				shell.AddCmd(&ishell.Cmd{
					Name: fmt.Sprintf("%s.%s", appname, k.String("domain")),
					Func: func(c *ishell.Context) {
						a, err := app.NewApp(filepath.Join(rootDir, appname), fmt.Sprintf("https://%s.%s/", appname, k.String("domain")), k.StringMap("env"))
						if err != nil {
							c.Println(err)
							return
						}
						a.Env["FORCE_COLOR"] = "1"

						output, err := a.Output(c.Args...)
						if err != nil {
							c.Err(err)
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
