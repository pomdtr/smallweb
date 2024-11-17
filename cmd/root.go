package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/mattn/go-isatty"

	"github.com/pomdtr/smallweb/build"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

var (
	k = koanf.New(".")
)

func NewCmdRoot(changelog string) *cobra.Command {
	defaultProvider := confmap.Provider(map[string]interface{}{
		"addr":   ":7777",
		"domain": "localhost",
	}, "")

	envProvider := env.Provider("SMALLWEB_", ".", func(s string) string {
		if s == "SMALLWEB_DIR" {
			return ""
		}

		key := strings.TrimPrefix(s, "SMALLWEB_")
		return strings.Replace(strings.ToLower(key), "_", ".", -1)
	})

	rootDir := utils.RootDir()
	configPath := filepath.Join(rootDir, ".smallweb", "config.json")
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

	cmd := &cobra.Command{
		Use:                "smallweb",
		Short:              "Host websites from your internet folder",
		Version:            build.Version,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		ValidArgsFunction:  completePlugins(),
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
				return cmd.Help()
			}

			if args[0] == "--version" || args[0] == "-v" {
				fmt.Fprintln(os.Stdout, build.Version)
				return nil
			}

			return fmt.Errorf("unknown command: %s", args[0])
		},
	}

	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdUpgrade())
	cmd.AddCommand(NewCmdUp())
	cmd.AddCommand(NewCmdService())
	cmd.AddCommand(NewCmdConfig())
	cmd.AddCommand(NewCmdCreate())
	cmd.AddCommand(NewCmdOpen())
	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdRename())
	cmd.AddCommand(NewCmdDelete())
	cmd.AddCommand(NewCmdFetch())
	cmd.AddCommand(NewCmdCron())
	cmd.AddCommand(NewCmdLogs())
	cmd.AddCommand(NewCmdSync())
	cmd.AddCommand(&cobra.Command{
		Use:   "changelog",
		Short: "Show the changelog",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Println(changelog)
				return nil
			}

			out, err := glamour.Render(changelog, "dark")
			if err != nil {
				return fmt.Errorf("failed to render changelog: %w", err)
			}

			fmt.Println(out)
			return nil
		},
	})

	for _, pluginDir := range utils.PluginDirs() {
		entries, err := os.ReadDir(pluginDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			plugin := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			cmd.AddCommand(&cobra.Command{
				Use:                strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
				Short:              fmt.Sprintf("Run the %s plugin", plugin),
				DisableFlagParsing: true,
				RunE: func(cmd *cobra.Command, args []string) error {
					entrypoint := filepath.Join(pluginDir, entry.Name())

					if ok, err := isExecutable(entrypoint); err != nil {
						return fmt.Errorf("failed to check if plugin is executable: %w", err)
					} else if !ok {
						if err := os.Chmod(entrypoint, 0755); err != nil {
							return fmt.Errorf("failed to make plugin executable: %w", err)
						}
					}

					command := exec.Command(entrypoint, args...)
					command.Env = os.Environ()
					command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_VERSION=%s", build.Version))
					command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_DIR=%s", rootDir))
					command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_DOMAIN=%s", k.String("domain")))

					command.Stdin = os.Stdin
					command.Stdout = os.Stdout
					command.Stderr = os.Stderr

					cmd.SilenceErrors = true
					return command.Run()
				},
			})

		}
	}

	return cmd
}

func completePlugins() func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var plugins []string
		for _, pluginDir := range utils.PluginDirs() {
			entries, err := os.ReadDir(pluginDir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				entrypoint := filepath.Join(pluginDir, entry.Name())
				plugin := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				plugins = append(plugins, fmt.Sprintf("%s\t%s", plugin, utils.AddTilde(entrypoint)))
			}
		}

		return plugins, cobra.ShellCompDirectiveDefault
	}
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

	return "vi"
}
