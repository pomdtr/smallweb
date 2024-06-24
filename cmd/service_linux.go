//go:build linux

package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

//go:embed service/smallweb.service
var serviceConfigBytes []byte
var serviceConfig = template.Must(template.New("service").Parse(string(serviceConfigBytes)))

func NewCmdService() *cobra.Command {
	cmd := &cobra.Command{
		Use: "service",
	}

	cmd.AddCommand(NewCmdServiceInstall())
	cmd.AddCommand(NewCmdServiceUninstall())
	cmd.AddCommand(NewCmdServiceLog())
	cmd.AddCommand(NewCmdServiceStatus())
	return cmd
}

func NewCmdServiceInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use: "install",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDenoIfRequired(); err != nil {
				return err
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			servicePath := filepath.Join(homeDir, ".config", "systemd", "user", "smallweb.service")
			if worker.FileExists(servicePath) {
				return fmt.Errorf("service already installed")
			}

			if err := os.MkdirAll(filepath.Dir(servicePath), 0755); err != nil {
				return fmt.Errorf("failed to create service directory: %v", err)
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %v", err)
			}

			f, err := os.Create(servicePath)
			if err != nil {
				return fmt.Errorf("failed to create service file: %v", err)
			}
			defer f.Close()

			if err := serviceConfig.Execute(f, map[string]string{
				"ExecPath": execPath,
			}); err != nil {
				return fmt.Errorf("failed to write service file: %v", err)
			}

			// Reload the systemd manager configuration
			if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
				return fmt.Errorf("failed to reload systemd manager configuration: %v", err)
			}

			// Enable the service to start on boot
			if err := exec.Command("systemctl", "--user", "enable", "smallweb.service").Run(); err != nil {
				return fmt.Errorf("failed to enable service: %v", err)
			}

			// Start the service immediately
			if err := exec.Command("systemctl", "--user", "start", "smallweb.service").Run(); err != nil {
				return fmt.Errorf("failed to start service: %v", err)
			}

			return nil
		},
	}
	return cmd
}

func NewCmdServiceUninstall() *cobra.Command {
	cmd := &cobra.Command{
		Use: "uninstall",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			servicePath := filepath.Join(homeDir, ".config", "systemd", "user", "smallweb.service")
			if !worker.FileExists(servicePath) {
				return fmt.Errorf("service not installed")
			}

			// Stop the service if it is running
			if err := exec.Command("systemctl", "--user", "stop", "smallweb.service").Run(); err != nil {
				return fmt.Errorf("failed to stop service: %v", err)
			}

			// Disable the service so it doesn't start on boot
			if err := exec.Command("systemctl", "--user", "disable", "smallweb.service").Run(); err != nil {
				return fmt.Errorf("failed to disable service: %v", err)
			}

			if err := os.Remove(servicePath); err != nil {
				return fmt.Errorf("failed to remove service file: %v", err)
			}

			// Reload the systemd manager configuration
			if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
				return fmt.Errorf("failed to reload systemd manager configuration: %v", err)
			}

			return nil
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
		Aliases: []string{"logs"},
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			servicePath := filepath.Join(homeDir, ".config", "systemd", "user", "smallweb.service")
			if !worker.FileExists(servicePath) {
				return fmt.Errorf("service not installed")
			}

			logCmdArgs := []string{"--user", "--user-unit", "smallweb.service"}
			if flags.follow {
				logCmdArgs = append(logCmdArgs, "-f")
			}

			logCmd := exec.Command("journalctl", logCmdArgs...)
			logCmd.Stdout = os.Stdout
			logCmd.Stderr = os.Stderr
			if err := logCmd.Run(); err != nil {
				return fmt.Errorf("failed to execute journalctl: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flags.follow, "follow", "f", false, "Follow log output")
	return cmd
}

func NewCmdServiceStatus() *cobra.Command {
	return &cobra.Command{
		Use: "status",
		RunE: func(cmd *cobra.Command, args []string) error {
			statusCmd := exec.Command("systemctl", "--user", "status", "smallweb.service")
			statusCmd.Stdout = os.Stdout
			statusCmd.Stderr = os.Stderr
			if err := statusCmd.Run(); err != nil {
				return fmt.Errorf("failed to get service status: %v", err)
			}
			return nil
		},
	}
}
