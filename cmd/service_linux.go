//go:build linux

package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pomdtr/smallweb/utils"
)

//go:embed embed/smallweb.service
var serviceConfigBytes []byte
var serviceConfig = template.Must(template.New("service").Parse(string(serviceConfigBytes)))
var servicePath = filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "smallweb.service")

func InstallService(args []string) error {
	if utils.FileExists(servicePath) {
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

	if err := serviceConfig.Execute(f, map[string]any{
		"ExecPath":    execPath,
		"SmallwebDir": k.String("dir"),
		"Args":        strings.Join(args, " "),
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
}

func StartService() error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("systemctl", "--user", "start", "smallweb.service").Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func StopService() error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("systemctl", "--user", "stop", "smallweb.service").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	return nil
}

func RestartService() error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("systemctl", "--user", "restart", "smallweb.service").Run(); err != nil {
		return fmt.Errorf("failed to restart service: %v", err)
	}

	return nil
}

func UninstallService() error {
	if !utils.FileExists(servicePath) {
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
}

func PrintServiceLogs(follow bool) error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	logCmdArgs := []string{"--user", "--user-unit", "smallweb.service"}
	if follow {
		logCmdArgs = append(logCmdArgs, "-f")
	}

	logCmd := exec.Command("journalctl", logCmdArgs...)
	logCmd.Stdout = os.Stdout
	logCmd.Stderr = os.Stderr
	if err := logCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute journalctl: %v", err)
	}

	return nil
}

func ViewServiceStatus() error {
	statusCmd := exec.Command("systemctl", "--user", "status", "smallweb.service")
	statusCmd.Stdout = os.Stdout
	statusCmd.Stderr = os.Stderr
	if err := statusCmd.Run(); err != nil {
		return fmt.Errorf("failed to get service status: %v", err)
	}
	return nil
}
