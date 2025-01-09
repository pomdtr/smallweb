package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewCmdSync() *cobra.Command {
	var flags struct {
		name string
	}

	cmd := &cobra.Command{
		Use:   "sync <remote>",
		Short: "Sync the smallweb config with the filesystem",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkMutagen()
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alpha := args[0]
			beta := k.String("dir")

			command := exec.Command("mutagen", "sync", "create", "--ignore=node_modules,.DS_Store", "--ignore-vcs", "--mode=two-way-resolved", alpha, beta)

			if flags.name != "" {
				command.Args = append(command.Args, "--name", flags.name)
			}

			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if err := command.Run(); err != nil {
				return fmt.Errorf("failed to run mutagen: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "The name of the sync session")
	return cmd
}

func checkMutagen() error {
	_, err := exec.LookPath("mutagen")
	if err != nil {
		return fmt.Errorf("could not find mutagen executable")
	}

	return nil
}
