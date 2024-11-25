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
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync the smallweb config with the filesystem",
		Long: "Wrapper around the mutagen command.\n\n" +
			"It creates a two-way sync between the smallweb directory on your local machine and the smallweb directory on a remote machine. ",
	}

	cmd.AddCommand(NewCmdSyncCreate())
	cmd.AddCommand(NewCmdSyncDelete())
	cmd.AddCommand(NewCmdSyncStatus())
	cmd.AddCommand(NewCmdSyncDaemon())

	return cmd
}

func syncName() string {
	return strings.Replace(k.String("domain"), ".", "-", -1)
}

func checkMutagen(cmd *cobra.Command, args []string) error {
	_, err := exec.LookPath("mutagen")
	if err != nil {
		return fmt.Errorf("could not find mutagen executable")
	}

	return nil
}

func NewCmdSyncCreate() *cobra.Command {
	return &cobra.Command{
		Use:     "create <remote> [dir]",
		Short:   "Sync the smallweb config with the filesystem",
		PreRunE: checkMutagen,
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			remote := args[0]
			remoteDir, err := getRemoteDir(args[0])
			if err != nil {
				return fmt.Errorf("could not get remote dir: %v", err)
			}

			alpha := fmt.Sprintf("%s:%s", remote, remoteDir)

			var beta string
			if len(args) > 1 {
				beta = args[1]
			} else {
				beta = utils.RootDir()
			}

			return mutagen("sync", "create", fmt.Sprintf("--name=%s", syncName()), "--ignore=node_modules,.DS_Store", "--ignore-vcs", "--mode=two-way-resolved", alpha, beta)
		},
	}
}

func NewCmdSyncDelete() *cobra.Command {
	return &cobra.Command{
		Use:     "delete",
		Short:   "Terminate the smallweb sync",
		PreRunE: checkMutagen,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("sync", "terminate", syncName())
		},
	}
}

func NewCmdSyncStatus() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Monitor the smallweb sync",
		PreRunE: checkMutagen,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("sync", "monitor", syncName())
		},
	}
}

func NewCmdSyncDaemon() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service",
		PreRunE: checkMutagen,
		Short:   "Manage the smallweb sync service",
	}

	cmd.AddCommand(NewCmdSyncServiceStart())
	cmd.AddCommand(NewCmdSyncServiceStop())
	cmd.AddCommand(NewCmdSyncServiceInstall())
	cmd.AddCommand(NewCmdSyncServiceUninstall())

	return cmd
}

func NewCmdSyncServiceStart() *cobra.Command {
	return &cobra.Command{
		Use:     "start",
		Short:   "Start the smallweb sync daemon",
		PreRunE: checkMutagen,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("daemon", "start")
		},
	}
}

func NewCmdSyncServiceStop() *cobra.Command {
	return &cobra.Command{
		Use:     "stop",
		Short:   "Stop the smallweb sync daemon",
		PreRunE: checkMutagen,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("mutagen", "daemon", "stop")
		},
	}
}

func NewCmdSyncServiceInstall() *cobra.Command {
	return &cobra.Command{
		Use:     "install",
		Short:   "Register the smallweb sync daemon",
		PreRunE: checkMutagen,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("mutagen", "daemon", "register")
		},
	}
}

func NewCmdSyncServiceUninstall() *cobra.Command {
	return &cobra.Command{
		Use:     "uninstall",
		Short:   "Unregister the smallweb sync daemon",
		PreRunE: checkMutagen,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("mutagen", "daemon", "unregister")
		},
	}
}

func mutagen(args ...string) error {
	execPath, err := exec.LookPath("mutagen")
	if err != nil {
		return fmt.Errorf("could not find mutagen executable")
	}

	command := exec.Command(execPath, args...)

	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}

func getRemoteDir(remote string) (string, error) {
	command := exec.Command("ssh", remote, "smallweb", "config", "dir")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("could not get remote dir: %w", err)
	}

	return strings.TrimRight(string(output), "\n"), nil
}
