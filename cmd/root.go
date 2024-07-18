package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/adrg/xdg"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"

	"github.com/knadh/koanf/v2"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

const (
	CoreGroupID      = "core"
	ExtensionGroupID = "extension"
)

var (
	cachedUpgradePath = filepath.Join(xdg.CacheHome, "smallweb", "latest_version")
)

// Global koanf instance. Use "." as the key path delimiter. This can be "/" or any character.
var k = koanf.New(".")

func expandDomains(domains map[string]string) map[string]string {
	for key, value := range domains {
		domain, err := utils.ExpandTilde(value)
		if err != nil {
			domains[key] = value
		}

		domains[key] = domain
	}

	return domains
}

func NewCmdRoot(version string) *cobra.Command {
	var configDir string
	if env, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		configDir = filepath.Join(env, "smallweb", "config.json")
	} else {
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "smallweb")
	}

	defaultProvider := confmap.Provider(map[string]interface{}{
		"host": "127.0.0.1",
		"port": 7777,
		"domains": map[string]string{
			"*.localhost": "~/localhost/*",
		},
		"env": map[string]string{
			"DENO_TLS_CA_STORE": "system",
		},
	}, "")

	var configPath string
	if env, ok := os.LookupEnv("SMALLWEB_CONFIG"); ok {
		configPath = env
	} else if utils.FileExists(filepath.Join(configDir, "config.jsonc")) {
		configPath = filepath.Join(configDir, "config.jsonc")
	} else {
		configPath = filepath.Join(configDir, "config.json")
	}

	fileProvider := file.Provider(configPath)
	envProvider := env.Provider("SMALLWEB_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "SMALLWEB_")), "_", ".", -1)
	})

	k.Load(defaultProvider, nil)
	k.Load(fileProvider, utils.ConfigParser())
	k.Load(envProvider, nil)

	fileProvider.Watch(func(event interface{}, err error) {
		k = koanf.New(".")
		k.Load(defaultProvider, nil)
		k.Load(fileProvider, utils.ConfigParser())
		k.Load(envProvider, nil)
	})

	cmd := &cobra.Command{
		Use:     "smallweb",
		Short:   "Host websites from your internet folder",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if version == "dev" {
				return nil
			}

			if stat, err := os.Stat(cachedUpgradePath); err == nil && stat.ModTime().Add(24*time.Hour).After(time.Now()) {
				return nil
			}

			v, err := fetchLatestVersion()
			if err != nil {
				cmd.PrintErrln("failed to get version information:", err)
				return nil
			}

			if err := os.MkdirAll(filepath.Dir(cachedUpgradePath), 0755); err != nil {
				cmd.PrintErrln("failed to create upgrade cache directory:", err)
				return nil
			}

			if err := os.WriteFile(cachedUpgradePath, []byte(v.String()), 0644); err != nil {
				cmd.PrintErrln("failed to write upgrade cache:", err)
				return nil
			}

			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if version == "dev" {
				return
			}

			current, err := semver.NewVersion(cmd.Root().Version)
			if err != nil {
				cmd.PrintErrln("failed to parse current version:", err)
				return
			}

			if !utils.FileExists(cachedUpgradePath) {
				return
			}

			var latest *semver.Version
			data, err := os.ReadFile(cachedUpgradePath)
			if err != nil {
				cmd.PrintErrln("failed to read upgrade cache:", err)
				return
			}

			v, err := semver.NewVersion(string(data))
			if err != nil {
				cmd.PrintErrln("failed to parse cached version:", err)
				return
			}
			latest = v

			if latest.GreaterThan(current) {
				cmd.PrintErrln()
				cmd.PrintErrln("A new smallweb version is available:", latest.String())
				cmd.PrintErrln("Run `smallweb upgrade` to upgrade to the latest version")
			}
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

	cmd.AddCommand(NewCmdUp())
	cmd.AddCommand(NewCmdService())
	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdInit())
	cmd.AddCommand(NewCmdCron())
	cmd.AddCommand(NewCmdOpen())
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
