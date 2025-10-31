package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"

	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/build"
	"github.com/pomdtr/smallweb/internal/utils"
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
				return cmd.Help()
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
	rootCmd.AddCommand(NewCmdConfig())
	rootCmd.AddCommand(NewCmdLink())
	rootCmd.AddCommand(NewCmdSSHEntrypoint())
	rootCmd.AddCommand(NewCmdCreate())

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

	apps, err := app.ListApps(k.String("dir"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return apps, cobra.ShellCompDirectiveNoFileComp
}
