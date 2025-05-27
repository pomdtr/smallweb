package cmd

import (
	"fmt"
	"os"

	"github.com/abiosoft/ishell/v2"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/worker"
	"github.com/spf13/cobra"
)

func NewCmdShell() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Start a shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ishell.New()
			shell.SetPrompt("> ")

			appnames, err := app.LookupApps(k.String("dir"))
			if err != nil {
				return fmt.Errorf("failed to lookup apps: %w", err)
			}

			shell.DeleteCmd("exit")
			shell.DeleteCmd("help")
			shell.DeleteCmd("clear")

			shell.AddCmd(&ishell.Cmd{
				Name: "/help",
				Help: "display help",
				Func: func(c *ishell.Context) {
					c.Println(c.HelpText())
				},
			})

			shell.AddCmd(&ishell.Cmd{
				Name: "/clear",
				Help: "clear the screen",
				Func: func(c *ishell.Context) {
					err := c.ClearScreen()
					if err != nil {
						c.Err(err)
					}
				},
			})

			shell.AddCmd(&ishell.Cmd{
				Name: "/exit",
				Help: "exit the program",
				Func: func(c *ishell.Context) {
					c.Stop()
				},
			})

			shell.AddCmd(&ishell.Cmd{
				Name:    "/ls",
				Aliases: []string{"/list"},
				Help:    "list available apps",
				Func: func(c *ishell.Context) {
					if len(appnames) == 0 {
						cmd.PrintErrln("No apps found.")
						return
					}

					for _, appname := range appnames {
						cmd.Println(appname)
					}
				},
			})

			shell.NotFound(func(c *ishell.Context) {
				c.Err(fmt.Errorf("command not found: %s", c.Args[0]))
			})

			for _, appname := range appnames {
				shell.AddCmd(&ishell.Cmd{
					Name: appname,
					Help: fmt.Sprintf("run %s app", appname),
					Func: func(c *ishell.Context) {
						a, err := app.LoadApp(appname, k.String("dir"), k.String("domain"))
						if err != nil {
							c.Err(fmt.Errorf("failed to load app %s: %w", appname, err))
							return
						}

						wk := worker.NewWorker(a, k.Bool(fmt.Sprintf("apps.%s.admin", a.Name)), nil)

						command, err := wk.Command(cmd.Context(), c.Args)
						if err != nil {
							c.Err(fmt.Errorf("failed to create command for app %s: %w", appname, err))
							return
						}

						c.Print()

						command.Stdout = os.Stdout
						command.Stderr = os.Stderr

						command.Run()
					},
				})

			}

			shell.Run()
			return nil
		},
	}

	return cmd
}
