package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	CoreGroupID      = "core"
	ExtensionGroupID = "extension"
)

func NewCmdRoot(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "smallweb",
		Short:        "Host websites from your internet folder",
		Version:      version,
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
	cmd.AddCommand(NewCmdDump())
	cmd.AddCommand(NewCmdDocs())
	cmd.AddCommand(NewCmdCreate())
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
