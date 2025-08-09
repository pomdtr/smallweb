package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell/v2"
	"github.com/abiosoft/readline"
	"github.com/adrg/xdg"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/build"
	"github.com/pomdtr/smallweb/internal/utils"
	"github.com/pomdtr/smallweb/internal/worker"
	"github.com/spf13/cobra"
)

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return fmt.Sprintf("exit with code %d", e)
}

var (
	k = koanf.New(".")
)

func findSmallwebDir() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for filepath.Dir(currentDir) != currentDir {
		if _, err := os.Stat(filepath.Join(currentDir, ".smallweb")); err == nil {
			return currentDir
		}

		currentDir = filepath.Dir(currentDir)
	}

	return ""
}

var envProvider = env.ProviderWithValue("SMALLWEB_", ".", func(s string, v string) (string, interface{}) {
	switch s {
	case "SMALLWEB_DIR":
		return "dir", v
	}

	return "", nil
})

type fakeReadCloser struct {
	io.Reader
}

func (f fakeReadCloser) Close() error {
	return nil
}

func NewCmdRoot() *cobra.Command {
	_ = k.Load(envProvider, nil)
	rootCmd := &cobra.Command{
		Use:           "smallweb",
		Short:         "Host websites from your internet folder",
		Version:       build.Version,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
			_ = k.Load(confmap.Provider(map[string]interface{}{
				"dir": findSmallwebDir(),
			}, "."), nil)

			_ = k.Load(envProvider, nil)
			_ = k.Load(flagProvider, nil)

			configPath := utils.FindConfigPath(k.String("dir"))
			fileProvider := file.Provider(configPath)

			_ = k.Load(fileProvider, utils.ConfigParser())
			_ = k.Load(envProvider, nil)
			_ = k.Load(flagProvider, nil)

			return nil
		},
		ValidArgsFunction: completeCommands,
		SilenceUsage:      true,
		Args:              cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
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
					Name:    "/list",
					Aliases: []string{"/ls"},
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
							a, err := app.LoadApp(filepath.Join(k.String("dir"), appname))
							if err != nil {
								c.Err(fmt.Errorf("failed to load app %s: %w", appname, err))
								return
							}

							wk := worker.NewWorker(a, nil)

							command, err := wk.Command(cmd.Context(), c.Args)
							if err != nil {
								c.Err(fmt.Errorf("failed to create command for app %s: %w", appname, err))
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
			}

			if env, ok := os.LookupEnv("SMALLWEB_DISABLE_CUSTOM_COMMANDS"); ok {
				if disableCustomCommands, _ := strconv.ParseBool(env); disableCustomCommands {
					return fmt.Errorf("unknown command \"%s\" for \"smallweb\"", args[0])
				}
			}

			for _, pluginDir := range []string{
				filepath.Join(k.String("dir"), ".smallweb", "commands"),
				filepath.Join(xdg.ConfigHome, "smallweb", "commands"),
			} {
				entries, err := os.ReadDir(pluginDir)
				if err != nil {
					continue
				}

				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}

					plugin := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
					if plugin != args[0] {
						continue
					}

					entrypoint := filepath.Join(pluginDir, entry.Name())

					if ok, err := isExecutable(entrypoint); err != nil {
						return fmt.Errorf("failed to check if plugin is executable: %w", err)
					} else if !ok {
						if err := os.Chmod(entrypoint, 0755); err != nil {
							return fmt.Errorf("failed to make plugin executable: %w", err)
						}
					}

					command := exec.Command(entrypoint, args[1:]...)
					command.Env = os.Environ()
					command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_VERSION=%s", build.Version))
					command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_DIR=%s", k.String("dir")))
					command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_DOMAIN=%s", k.String("domain")))

					command.Stdin = os.Stdin
					command.Stdout = cmd.OutOrStdout()
					command.Stderr = cmd.ErrOrStderr()

					cmd.SilenceErrors = true
					return command.Run()
				}
			}

			return fmt.Errorf("unknown command \"%s\" for \"smallweb\"", args[0])
		},
	}

	rootCmd.PersistentFlags().String("dir", "", "The root directory for smallweb")
	rootCmd.PersistentFlags().String("domain", "", "The domain for smallweb")
	rootCmd.Flags().SetInterspersed(false)

	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdDocs())
	rootCmd.AddCommand(NewCmdUp())
	rootCmd.AddCommand(NewCmdDoctor())
	rootCmd.AddCommand(NewCmdList())
	rootCmd.AddCommand(NewCmdCrons())
	rootCmd.AddCommand(NewCmdInit())
	rootCmd.AddCommand(NewCmdOpen())
	rootCmd.AddCommand(NewCmdConfig())
	rootCmd.AddCommand(NewCmdLink())
	rootCmd.AddCommand(NewCmdGitReceivePack())
	rootCmd.AddCommand(NewCmdGitUploadPack())
	rootCmd.AddCommand(NewCmdCreate())

	if _, ok := os.LookupEnv("SMALLWEB_DISABLE_COMPLETIONS"); ok {
		rootCmd.CompletionOptions.DisableDefaultCmd = true
	}

	if env, ok := os.LookupEnv("SMALLWEB_DISABLED_COMMANDS"); ok {
		disabledCommands := strings.Split(env, ",")
		for _, commandName := range disabledCommands {
			if commandName == "completion" {
				rootCmd.CompletionOptions.DisableDefaultCmd = true
				continue // Skip disabling the completion command
			}

			if command, ok := GetCommand(rootCmd, commandName); ok {
				rootCmd.RemoveCommand(command)
			}
		}
	}

	return rootCmd
}

func GetCommand(cmd *cobra.Command, name string) (*cobra.Command, bool) {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c, true
		}
	}

	return nil, false
}

func isExecutable(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode().Perm()&0111 != 0, nil
}

func completeCommands(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
	_ = k.Load(flagProvider, nil)

	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	var completions []string
	for _, dir := range []string{
		filepath.Join(k.String("dir"), ".smallweb", "commands"),
		filepath.Join(xdg.ConfigHome, "smallweb", "commands"),
	} {

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			completions = append(completions, fmt.Sprintf("%s\tCustom command", name))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeApp(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
	_ = k.Load(flagProvider, nil)

	apps, err := app.LookupApps(k.String("dir"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return apps, cobra.ShellCompDirectiveNoFileComp
}
