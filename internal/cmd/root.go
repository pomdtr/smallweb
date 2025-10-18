package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"

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
			configPath, err := utils.FindConfigPath()
			if err != nil {
				return fmt.Errorf("failed to find config file: %w", err)
			}

			fileProvider := file.Provider(configPath)
			flagProvider := posflag.Provider(cmd.Root().Flags(), ".", k)

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}

			_ = k.Load(confmap.Provider(map[string]interface{}{
				"dir": filepath.Join(homeDir, "smallweb"),
			}, "."), nil)

			_ = k.Load(fileProvider, utils.ConfigParser())
			_ = k.Load(envProvider, nil)
			_ = k.Load(flagProvider, nil)

			if k.String("domain") == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current working directory: %w", err)
				}

				relpath, err := filepath.Rel(k.String("dir"), cwd)
				if err != nil {
					return fmt.Errorf("failed to get relative path: %w", err)
				}

				if strings.HasPrefix(relpath, "..") {
					return nil
				}

				parts := strings.Split(relpath, string(os.PathSeparator))
				if len(parts) < 1 {
					return nil
				}

				domain := parts[0]
				if !strings.HasPrefix(domain, ".") {
					k.Set("domain", parts[0])
				}
			}

			return nil
		},
		ValidArgsFunction: completeCommands,
		SilenceUsage:      true,
		Args:              cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			configDir, err := os.UserConfigDir()
			if err != nil {
				return fmt.Errorf("failed to get user config directory: %w", err)
			}

			commandDir := filepath.Join(configDir, "commands")

			entries, err := os.ReadDir(commandDir)
			if err != nil {
				return nil
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				plugin := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				if plugin != args[0] {
					continue
				}

				entrypoint := filepath.Join(commandDir, entry.Name())

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

				command.Stdin = os.Stdin
				command.Stdout = cmd.OutOrStdout()
				command.Stderr = cmd.ErrOrStderr()

				cmd.SilenceErrors = true
				return command.Run()
			}

			return fmt.Errorf("unknown command \"%s\" for \"smallweb\"", args[0])
		},
	}

	rootCmd.PersistentFlags().String("dir", "", "The root directory for smallweb")
	rootCmd.PersistentFlags().String("domain", "", "")

	rootCmd.RegisterFlagCompletionFunc("domain", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
		_ = k.Load(flagProvider, nil)

		domains, err := ListDomains(k.String("dir"))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return domains, cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.Flags().SetInterspersed(false)

	rootCmd.AddCommand(NewCmdShell())
	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdDocs())
	rootCmd.AddCommand(NewCmdUp())
	rootCmd.AddCommand(NewCmdDoctor())
	rootCmd.AddCommand(NewCmdList())
	rootCmd.AddCommand(NewCmdCrons())
	rootCmd.AddCommand(NewCmdOpen())
	rootCmd.AddCommand(NewCmdConfig())
	rootCmd.AddCommand(NewCmdLink())
	rootCmd.AddCommand(NewCmdSSH())
	rootCmd.AddCommand(NewCmdCreate())
	rootCmd.AddCommand(NewCmdInit())

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

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, err := os.ReadDir(filepath.Join(configDir, "commands"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		completions = append(completions, fmt.Sprintf("%s\tCustom command", name))
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeApp(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	flagProvider := posflag.Provider(cmd.Root().PersistentFlags(), ".", k)
	_ = k.Load(flagProvider, nil)

	apps, err := ListApps(k.String("dir"), k.String("domain"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return apps, cobra.ShellCompDirectiveNoFileComp
}
