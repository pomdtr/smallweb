package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func NewCmdSync() *cobra.Command {
	return &cobra.Command{
		Use:   "sync <remote>",
		Short: "Sync the smallweb config with the filesystem",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkMutagen()
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			remote := args[0]
			localDir := k.String("dir")
			syncName := strings.Replace(k.String("domain"), ".", "-", -1)
			command := exec.Command("mutagen", "sync", "create", fmt.Sprintf("--name=%s", syncName), "--ignore=node_modules,.DS_Store", "--ignore-vcs", "--mode=two-way-resolved", remote, localDir)

			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if err := command.Run(); err != nil {
				return fmt.Errorf("failed to run mutagen: %v", err)
			}

			return nil
		},
	}
}

func checkMutagen() error {
	_, err := exec.LookPath("mutagen")
	if err != nil {
		return fmt.Errorf("could not find mutagen executable")
	}

	return nil
}
