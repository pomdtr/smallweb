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

func NewCmdRoot(version string, changelog string) *cobra.Command {
	defaultProvider := confmap.Provider(map[string]interface{}{
		"addr": ":7777",
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
		Use:          "smallweb",
		Short:        "Host websites from your internet folder",
		Version:      version,
		SilenceUsage: true,
	}

	cmd.AddGroup(&cobra.Group{
		ID:    CoreGroupID,
		Title: "Core Commands",
	})

	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdTunnel())
	cmd.AddCommand(NewCmdUpgrade())
	cmd.AddCommand(NewCmdToken())
	cmd.AddCommand(NewCmdUp())
	cmd.AddCommand(NewCmdService())
	cmd.AddCommand(NewCmdConfig())
	cmd.AddCommand(NewCmdCreate())
	cmd.AddCommand(NewCmdAPI())
	cmd.AddCommand(NewCmdOpen())
	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdRename())
	cmd.AddCommand(NewCmdClone())
	cmd.AddCommand(NewCmdDelete())

	cmd.AddCommand(&cobra.Command{
		Use:     "changelog",
		Short:   "Show the changelog",
		GroupID: CoreGroupID,
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
				command.Env = os.Environ()
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

	return "vi"
}
