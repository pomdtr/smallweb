package cmd

import (
	_ "embed"
	"errors"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

func getServiceName(domain string) string {
	parts := strings.Split(domain, ".")
	slices.Reverse(parts)
	parts = append(parts, "server")
	return strings.Join(parts, ".")
}

func NewCmdService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage smallweb service",
	}

	cmd.AddCommand(NewCmdServiceInstall())
	cmd.AddCommand(NewCmdServiceUninstall())
	cmd.AddCommand(NewCmdServiceLogs())
	cmd.AddCommand(NewCmdServiceStatus())
	cmd.AddCommand(NewCmdServiceStart())
	cmd.AddCommand(NewCmdServiceStop())
	cmd.AddCommand(NewCmdServiceRestart())

	return cmd
}

func NewCmdServiceInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install [args...]",
		Short:   "Install smallweb as a service",
		PreRunE: requireDomain,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := InstallService(args); err != nil {
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
		Use:     "uninstall",
		Short:   "Uninstall smallweb service",
		PreRunE: requireDomain,
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

func NewCmdServiceStart() *cobra.Command {
	return &cobra.Command{
		Use:     "start",
		Short:   "Start smallweb service",
		PreRunE: requireDomain,
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
		Use:     "stop",
		Short:   "Stop smallweb service",
		PreRunE: requireDomain,
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
		Use:     "restart",
		Short:   "Restart smallweb service",
		PreRunE: requireDomain,
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
		PreRunE: requireDomain,
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
		Use:     "status",
		Short:   "View service status",
		PreRunE: requireDomain,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ViewServiceStatus()
		},
	}
}

func requireDomain(cmd *cobra.Command, args []string) error {
	if k.String("domain") == "" {
		return errors.New("missing domain")
	}
	return nil
}
