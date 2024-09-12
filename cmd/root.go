package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/abiosoft/readline"
	"github.com/adrg/xdg"
	"github.com/charmbracelet/glamour"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/mattn/go-isatty"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/database"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

const (
	CoreGroupID      = "core"
	ExtensionGroupID = "extension"
)

var (
	k = koanf.New(".")
)

type ExitError struct {
	Code    int
	Message string
}

func NewExitError(code int, message string) *ExitError {
	return &ExitError{Code: code}
}

func (e *ExitError) Error() string {
	return e.Message
}

func NewCmdRoot(version string, changelog string) *cobra.Command {
	dataHome := filepath.Join(xdg.DataHome, "smallweb")
	if err := os.MkdirAll(dataHome, 0755); err != nil {
		fmt.Println("failed to create data directory:", err)
	}

	db, err := database.OpenDB(filepath.Join(dataHome, "smallweb.db"))
	if err != nil {
		fmt.Println("failed to open database:", err)
		return nil
	}

	defaultProvider := confmap.Provider(map[string]interface{}{
		"host":   "127.0.0.1",
		"dir":    "~/smallweb",
		"editor": findEditor(),
		"domain": "localhost",
		"env": map[string]string{
			"DENO_TLS_CA_STORE": "system",
		},
	}, "")

	envProvider := env.Provider("SMALLWEB_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "SMALLWEB_")), "_", ".", -1)
	})

	configPath := findConfigPath()
	fileProvider := file.Provider(configPath)
	fileProvider.Watch(func(event interface{}, err error) {
		k = koanf.New(".")
		k.Load(defaultProvider, nil)
		k.Load(fileProvider, utils.ConfigParser())
		k.Load(envProvider, nil)
	})

	k.Load(defaultProvider, nil)
	k.Load(fileProvider, utils.ConfigParser())
	k.Load(envProvider, nil)

	if k.String("remote") != "" {
		cmd := cobra.Command{
			Use:                "smallweb",
			Short:              "Proxy args to remote smallweb server",
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				sshArgs := []string{"-t", "-o", "LogLevel=QUIET", k.String("remote"), "~/.local/bin/smallweb"}
				sshArgs = append(sshArgs, args...)
				command := exec.Command("ssh", sshArgs...)
				command.Stdin = os.Stdin
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				return command.Run()
			},
		}

		return &cmd
	}

	cmd := &cobra.Command{
		Use:           "smallweb",
		Short:         "Host websites from your internet folder",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isatty.IsTerminal(os.Stdin.Fd()) || os.Getenv("SMALLWEB") == "1" {
				return cmd.Help()
			}
			rootDir := utils.ExpandTilde(k.String("dir"))
			shell := ishell.NewWithConfig(&readline.Config{
				Prompt: "\033[32m$\033[0m ",
			})
			cacheDir := filepath.Join(xdg.DataHome, "smallweb", "history")
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create history directory: %w", err)
			}

			apps, err := app.ListApps(rootDir)
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			shell.DeleteCmd("exit")
			shell.DeleteCmd("help")
			shell.DeleteCmd("clear")

			for _, name := range apps {
				a, err := app.LoadApp(filepath.Join(rootDir, name))
				if err != nil {
					shell.Println("failed to load app:", err)
					continue
				}

				shell.AddCmd(&ishell.Cmd{
					Name: a.Name(),
					Func: func(c *ishell.Context) {
						executable, err := os.Executable()
						if err != nil {
							c.Err(err)
							return
						}

						cmd := exec.Command(executable, "run", a.Name())
						cmd.Env = os.Environ()
						cmd.Env = append(cmd.Env, "SMALLWEB=1")
						cmd.Args = append(cmd.Args, c.Args...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr

						if err := cmd.Run(); err != nil {
							c.Err(err)
						}
					},
				})
			}

			shell.SetHistoryPath(filepath.Join(cacheDir, "history"))

			shell.Run()
			return nil
		},
	}

	cmd.AddGroup(&cobra.Group{
		ID:    CoreGroupID,
		Title: "Core Commands",
	})

	cmd.AddCommand(NewCmdUp(db))
	cmd.AddCommand(NewCmdEdit())
	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdService())
	cmd.AddCommand(NewCmdOpen())
	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdCron())
	cmd.AddCommand(NewCmdVersion())
	cmd.AddCommand(NewCmdCreate())
	cmd.AddCommand(NewCmdToken(db))

	cmd.AddCommand(&cobra.Command{
		Use:   "changelog",
		Short: "Show the changelog",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := glamour.Render(changelog, "dark")
			if err != nil {
				return NewExitError(1, fmt.Sprintf("failed to render changelog: %v", err))
			}

			fmt.Println(out)
			return nil
		},
	})

	var extensions []string
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if !strings.HasPrefix(entry.Name(), "smallweb-") {
				continue
			}

			entrypoint := filepath.Join(dir, entry.Name())
			if ok, err := isExecutable(entrypoint); !ok || err != nil {
				continue
			}

			extensions = append(extensions, entrypoint)
		}
	}

	if len(extensions) == 0 {
		return cmd
	}

	cmd.AddGroup(&cobra.Group{
		ID:    ExtensionGroupID,
		Title: "Extension Commands",
	})

	for _, entrypoint := range extensions {
		name := strings.TrimPrefix(filepath.Base(entrypoint), "smallweb-")
		if HasCommand(cmd, name) {
			continue
		}

		cmd.AddCommand(&cobra.Command{
			Use:                name,
			Short:              fmt.Sprintf("Extension %s", name),
			GroupID:            ExtensionGroupID,
			DisableFlagParsing: true,
			SilenceErrors:      true,
			RunE: func(cmd *cobra.Command, args []string) error {
				command := exec.Command(entrypoint, args...)
				command.Stdin = os.Stdin
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				return command.Run()
			},
		})
	}

	return cmd
}

func HasCommand(cmd *cobra.Command, name string) bool {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return true
		}
	}
	return false
}

func isExecutable(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode().Perm()&0111 != 0, nil
}

func findEditor() string {
	if env, ok := os.LookupEnv("EDITOR"); ok {
		return env
	}

	return "vim -Z"
}
