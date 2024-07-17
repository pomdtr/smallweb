package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	CoreGroupID      = "core"
	ExtensionGroupID = "extension"
)

func expandTilde(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return strings.Replace(path, "~", home, 1), nil
	}
	return path, nil
}

func extractDomains(v *viper.Viper) map[string]string {
	domains := v.GetStringMapString("domains")
	for key, value := range domains {
		domain, err := expandTilde(value)
		if err != nil {
			domains[key] = value
		}

		domains[key] = domain
	}

	return domains
}

func NewCmdRoot(version string) *cobra.Command {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("json")
	v.AddConfigPath(filepath.Join(xdg.ConfigHome, "smallweb"))
	v.AddConfigPath("$HOME/.config/smallweb")
	v.SetEnvPrefix("SMALLWEB")
	v.AutomaticEnv()

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println(extractDomains(v))
	})

	v.SetDefault("host", "127.0.0.1")
	v.SetDefault("port", 7777)
	v.SetDefault("domains", map[string]string{
		"*.localhost": "~/localhost",
	})
	v.SetDefault("env", map[string]string{
		"DENO_TLS_CA_STORE": "system",
	})

	cmd := &cobra.Command{
		Use:     "smallweb",
		Short:   "Host websites from your internet folder",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := v.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return fmt.Errorf("failed to read config file: %v", err)
				}
			}

			return nil
		},
		SilenceUsage: true,
	}
	cmd.AddGroup(&cobra.Group{
		ID:    CoreGroupID,
		Title: "Core Commands",
	}, &cobra.Group{
		ID:    ExtensionGroupID,
		Title: "Extension Commands",
	})

	cmd.AddCommand(NewCmdUp(v))
	cmd.AddCommand(NewCmdService())
	cmd.AddCommand(NewCmdDump(v))
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdCreate())
	cmd.AddCommand(NewCmdCron(v))
	cmd.AddCommand(NewCmdOpen(v))
	cmd.AddCommand(NewCmdUpgrade())

	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "smallweb-") {
				entrypoint := filepath.Join(dir, entry.Name())
				// check if the entrypoint is executable
				if _, err := os.Stat(entrypoint); err != nil {
					continue
				}

				if ok, err := isExecutable(entrypoint); !ok || err != nil {
					continue
				}

				name := strings.TrimPrefix(entry.Name(), "smallweb-")
				if HasCommand(cmd, name) {
					continue
				}

				cmd.AddCommand(&cobra.Command{
					Use:                name,
					Short:              fmt.Sprintf("Extension %s", name),
					GroupID:            ExtensionGroupID,
					DisableFlagParsing: true,
					RunE: func(cmd *cobra.Command, args []string) error {
						command := exec.Command(entrypoint, args...)
						command.Stdin = os.Stdin
						command.Stdout = os.Stdout
						command.Stderr = os.Stderr
						return command.Run()
					},
				})
			}
		}
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
