package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/glamour"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/mattn/go-isatty"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/build"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

var (
	k = koanf.New(".")
)

func NewCmdRoot(changelog string) *cobra.Command {
	defaultProvider := confmap.Provider(map[string]interface{}{
		"dir": filepath.Join(os.Getenv("HOME"), "smallweb"),
	}, ".")
	envProvider := env.ProviderWithValue("SMALLWEB_", ".", func(s string, v string) (string, interface{}) {
		switch s {
		case "SMALLWEB_DIR":
			return "dir", v
		case "SMALLWEB_DOMAIN":
			return "domain", v
		case "SMALLWEB_CUSTOM_DOMAINS":
			customDomains := make(map[string]string)
			for _, entry := range strings.Split(v, ";") {
				parts := strings.Split(entry, "=")
				if len(parts) != 2 {
					continue
				}

				customDomains[parts[0]] = parts[1]
			}

			return "customDomains", customDomains
		}

		return "", nil
	})

	_ = k.Load(defaultProvider, nil)
	_ = k.Load(envProvider, nil)

	rootCmd := &cobra.Command{
		Use:     "smallweb",
		Short:   "Host websites from your internet folder",
		Version: build.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
			_ = k.Load(flagProvider, nil)

			configPath := filepath.Join(k.String("dir"), ".smallweb", "config.json")
			fileProvider := file.Provider(configPath)
			_ = k.Load(fileProvider, utils.ConfigParser())

			_ = fileProvider.Watch(func(event interface{}, err error) {
				k = koanf.New(".")
				_ = k.Load(defaultProvider, nil)
				_ = k.Load(envProvider, nil)
				_ = k.Load(posflag.Provider(cmd.PersistentFlags(), ".", k), nil)
				_ = k.Load(fileProvider, utils.ConfigParser())
			})

			return nil
		},
		ValidArgsFunction: completePlugins,
		SilenceUsage:      true,
		Args:              cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			if env, ok := os.LookupEnv("SMALLWEB_DISABLE_PLUGINS"); ok {
				if disablePlugins, _ := strconv.ParseBool(env); disablePlugins {
					return cmd.Help()
				}
			}

			for _, pluginDir := range []string{
				filepath.Join(k.String("dir"), ".smallweb", "plugins"),
				filepath.Join(xdg.DataHome, "smallweb", "plugins"),
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
					command.Stdout = os.Stdout
					command.Stderr = os.Stderr

					cmd.SilenceErrors = true
					return command.Run()
				}
			}

			return fmt.Errorf("unknown command \"%s\" for \"smallweb\"", args[0])
		},
	}

	rootCmd.PersistentFlags().String("dir", "", "The root directory for smallweb")

	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdDocs())
	rootCmd.AddCommand(NewCmdUpgrade())
	rootCmd.AddCommand(NewCmdUp())
	rootCmd.AddCommand(NewCmdService())
	rootCmd.AddCommand(NewCmdConfig())
	rootCmd.AddCommand(NewCmdDoctor())
	rootCmd.AddCommand(NewCmdOpen())
	rootCmd.AddCommand(NewCmdList())
	rootCmd.AddCommand(NewCmdFetch())
	rootCmd.AddCommand(NewCmdCrons())
	rootCmd.AddCommand(NewCmdInstall())
	rootCmd.AddCommand(NewCmdLogs())
	rootCmd.AddCommand(NewCmdSync())
	rootCmd.AddCommand(NewCmdSecrets())
	rootCmd.AddCommand(NewCmdLink())

	rootCmd.AddCommand(&cobra.Command{
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

	if env, ok := os.LookupEnv("SMALLWEB_DISABLED_COMMANDS"); ok {
		disabledCommands := strings.Split(env, ",")
		for _, commandName := range disabledCommands {
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

func completePlugins(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
	_ = k.Load(flagProvider, nil)

	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	var plugins []string
	for _, pluginDir := range []string{
		filepath.Join(k.String("dir"), ".smallweb", "plugins"),
		filepath.Join(xdg.DataHome, "smallweb", "plugins"),
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
			plugins = append(plugins, fmt.Sprintf("%s\tPlugin %s", plugin, plugin))
		}
	}

	return plugins, cobra.ShellCompDirectiveDefault
}

func completeApp(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
	_ = k.Load(flagProvider, nil)

	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	apps, err := app.ListApps(k.String("dir"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveDefault
	}

	return apps, cobra.ShellCompDirectiveDefault
}
