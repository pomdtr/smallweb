package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

func NewCmdService() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service",
		Short:   "Manage smallweb service",
		GroupID: CoreGroupID,
	}

	cmd.AddCommand(NewCmdServiceInstall())
	cmd.AddCommand(NewCmdServiceUninstall())
	cmd.AddCommand(NewCmdServiceEdit())
	cmd.AddCommand(NewCmdServiceLogs())
	cmd.AddCommand(NewCmdServiceStatus())
	cmd.AddCommand(NewCmdServiceStart())
	cmd.AddCommand(NewCmdServiceStop())
	cmd.AddCommand(NewCmdServiceRestart())

	return cmd
}

func NewCmdServiceInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install smallweb as a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := InstallService(); err != nil {
				return err
			}

			cmd.Println("Service installed successfully")
			return nil
		},
	}
	return cmd
}

func NewCmdServiceUninstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall smallweb service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := UninstallService(); err != nil {
				return err
			}

			cmd.Println("Service uninstalled successfully")
			return nil
		},
	}
	return cmd
}

func NewCmdServiceEdit() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit smallweb service configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			editor := findEditor()
			if err := os.MkdirAll(filepath.Dir(servicePath), 0755); err != nil {
				return err
			}

			editorArgs, err := shlex.Split(editor)
			if err != nil {
				return err
			}
			editorArgs = append(editorArgs, servicePath)

			command := exec.Command(editorArgs[0], editorArgs[1:]...)
			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if err := command.Run(); err != nil {
				return fmt.Errorf("failed to run editor: %v", err)
			}

			return nil
		},
	}
}

func NewCmdServiceStart() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start smallweb service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := StartService(); err != nil {
				return err
			}

			cmd.Println("Service started successfully")
			return nil
		},
	}
}

func NewCmdServiceStop() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop smallweb service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := StopService(); err != nil {
				return err
			}
			cmd.Println("Service stopped successfully")
			return nil
		},
	}
}

func NewCmdServiceRestart() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart smallweb service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := RestartService(); err != nil {
				return err
			}

			cmd.Println("Service restarted successfully")
			return nil
		},
	}
}

func NewCmdServiceLogs() *cobra.Command {
	var flags struct {
		follow bool
	}

	cmd := &cobra.Command{
		Use:     "logs",
		Short:   "Print service logs",
		Aliases: []string{"log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return PrintServiceLogs(flags.follow)
		},
	}

	cmd.Flags().BoolVarP(&flags.follow, "follow", "f", false, "Follow log output")
	return cmd
}

func NewCmdServiceStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "View service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ViewServiceStatus()
		},
	}
}
