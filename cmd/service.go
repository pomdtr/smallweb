package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
)

func NewCmdService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage smallweb service",
	}

	cmd.AddCommand(NewCmdServiceInstall())
	cmd.AddCommand(NewCmdServiceUninstall())
	cmd.AddCommand(NewCmdServiceLog())
	cmd.AddCommand(NewCmdServiceStatus())

	return cmd
}

func NewCmdServiceInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install smallweb as a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return InstallService()
		},
	}
	return cmd
}

func NewCmdServiceUninstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall smallweb service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UninstallService()
		},
	}
	return cmd

}

func NewCmdServiceLog() *cobra.Command {
	var flags struct {
		follow bool
	}

	cmd := &cobra.Command{
		Use:     "log",
		Short:   "Print service logs",
		Aliases: []string{"logs"},
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
