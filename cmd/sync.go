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
	cmd.AddCommand(NewCmdSyncTerminate())
	cmd.AddCommand(NewCmdSyncMonitor())
	cmd.AddCommand(NewCmdSyncPause())
	cmd.AddCommand(NewCmdSyncResume())
	cmd.AddCommand(NewCmdSyncDaemon())

	return cmd
}

func NewCmdSyncCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create <remote> [dir]",
		Short: "Sync the smallweb config with the filesystem",
		Args:  cobra.RangeArgs(1, 2),
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

			return mutagen("sync", "create", "--name=smallweb", "--ignore=node_modules", "--ignore-vcs", "--mode=two-way-resolved", alpha, beta)
		},
	}
}

func NewCmdSyncTerminate() *cobra.Command {
	return &cobra.Command{
		Use:   "terminate",
		Short: "Terminate the smallweb sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("sync", "terminate", "smallweb")
		},
	}
}

func NewCmdSyncMonitor() *cobra.Command {
	return &cobra.Command{
		Use:   "monitor",
		Short: "Monitor the smallweb sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("sync", "monitor", "smallweb")
		},
	}
}

func NewCmdSyncPause() *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause the smallweb sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("sync", "pause", "smallweb")
		},
	}
}

func NewCmdSyncResume() *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume the smallweb sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("sync", "resume", "smallweb")
		},
	}
}

func NewCmdSyncDaemon() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the smallweb sync daemon",
	}

	cmd.AddCommand(NewCmdSyncDaemonStart())
	cmd.AddCommand(NewCmdSyncDaemonStop())
	cmd.AddCommand(NewCmdSyncDaemonRegister())
	cmd.AddCommand(NewCmdSyncDaemonUnregister())

	return cmd
}

func NewCmdSyncDaemonStart() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the smallweb sync daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("daemon", "start")
		},
	}
}

func NewCmdSyncDaemonStop() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the smallweb sync daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("mutagen", "daemon", "stop")
		},
	}
}

func NewCmdSyncDaemonRegister() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register the smallweb sync daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutagen("mutagen", "daemon", "register")
		},
	}
}

func NewCmdSyncDaemonUnregister() *cobra.Command {
	return &cobra.Command{
		Use:   "unregister",
		Short: "Unregister the smallweb sync daemon",
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
