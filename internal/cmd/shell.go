package cmd

import (
	"fmt"
	"io"

	"github.com/abiosoft/ishell/v2"
	"github.com/abiosoft/readline"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/build"
	"github.com/pomdtr/smallweb/internal/worker"
	"github.com/spf13/cobra"
)

func NewCmdShell() *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Interactive shell to run apps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ishell.NewWithConfig(&readline.Config{
				Prompt:              "> ",
				ForceUseInteractive: true,
				FuncGetWidth: func() int {
					return 80 // Default terminal width
				},
				Stdin:  fakeReadCloser{cmd.InOrStdin()},
				Stdout: cmd.OutOrStdout(),
				Stderr: cmd.OutOrStdout(),
			})

			shell.AutoHelp(false)

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

			shell.NotFound(func(c *ishell.Context) {
				c.Err(fmt.Errorf("command not found: %s", c.Args[0]))
			})

			appnames, err := ListApps(k.String("dir"), k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to lookup apps: %w", err)
			}
			for _, name := range appnames {
				shell.AddCmd(&ishell.Cmd{
					Name: name,
					Help: fmt.Sprintf("run %s app", name),
					Func: func(c *ishell.Context) {
						a, err := app.LoadApp(k.String("dir"), k.String("domain"), c.Cmd.Name)
						if err != nil {
							c.Err(fmt.Errorf("failed to load app %s: %w", name, err))
							return
						}

						wk := worker.NewWorker(a, nil)

						command, err := wk.Command(cmd.Context(), c.Args)
						if err != nil {
							c.Err(fmt.Errorf("failed to create command for app %s: %w", name, err))
							return
						}

						c.Print()

						command.Stdout = cmd.OutOrStdout()
						command.Stderr = cmd.OutOrStdout()

						command.Run()
					},
				})

			}

			shell.Printf("Smallweb %s\n", build.Version)
			shell.Printf("use /help for a list of commands.\n")
			shell.Run()
			return nil

		},
	}
}

type fakeReadCloser struct {
	io.Reader
}

func (f fakeReadCloser) Close() error {
	return nil
}
