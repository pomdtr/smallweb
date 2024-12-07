package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdSync() *cobra.Command {
	return &cobra.Command{
		Use:   "sync <remote> <remote-dir>",
		Short: "Sync the smallweb config with the filesystem",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkMutagen()
		},
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			remote, remoteDir := args[0], args[1]
			beta := utils.RootDir
			syncName := strings.Replace(k.String("domain"), ".", "-", -1)
			command := exec.Command("mutagen", "sync", "create", fmt.Sprintf("--name=%s", syncName), "--ignore=node_modules,.DS_Store", "--ignore-vcs", "--mode=two-way-resolved", fmt.Sprintf("%s:%s", remote, remoteDir), beta)

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
