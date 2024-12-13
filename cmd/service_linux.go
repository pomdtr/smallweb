//go:build linux

package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pomdtr/smallweb/utils"
)

//go:embed embed/smallweb.service
var serviceConfigBytes []byte
var serviceConfig = template.Must(template.New("service").Parse(string(serviceConfigBytes)))

func getServicePath(uid int) string {
	if uid == 0 {
		return path.Join("/etc", "systemd", "system", "smallweb.service")
	}

	return path.Join(os.Getenv("HOME"), ".config", "systemd", "user", "smallweb.service")
}

func InstallService(args []string) error {
	uid := os.Getuid()
	servicePath := getServicePath(uid)
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

	user, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	if err := serviceConfig.Execute(f, map[string]any{
		"ExecPath":    execPath,
		"User":        user.Username,
		"SmallwebDir": k.String("dir"),
		"Args":        strings.Join(args, " "),
	}); err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			return fmt.Errorf("failed to reload systemd manager configuration: %v", err)
		}

		if err := exec.Command("systemctl", "enable", "smallweb").Run(); err != nil {
			return fmt.Errorf("failed to enable service: %v", err)
		}

		if err := exec.Command("systemctl", "start", "smallweb").Run(); err != nil {
			return fmt.Errorf("failed to start service: %v", err)
		}

		return nil
	}

	// Reload the systemd manager configuration
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd manager configuration: %v", err)
	}

	// Enable the service to start on boot
	if err := exec.Command("systemctl", "--user", "enable", "smallweb").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %v", err)
	}

	// Start the service immediately
	if err := exec.Command("systemctl", "--user", "start", "smallweb").Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func StartService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid)
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "start", "smallweb").Run(); err != nil {
			return fmt.Errorf("failed to start service: %v", err)
		}

		return nil
	}

	if err := exec.Command("systemctl", "--user", "start", "smallweb").Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func StopService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid)
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "stop", "smallweb").Run(); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}

		return nil
	}

	if err := exec.Command("systemctl", "--user", "stop", "smallweb").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	return nil
}

func RestartService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid)
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "restart", "smallweb").Run(); err != nil {
			return fmt.Errorf("failed to restart service: %v", err)
		}

		return nil
	}

	if err := exec.Command("systemctl", "--user", "restart", "smallweb").Run(); err != nil {
		return fmt.Errorf("failed to restart service: %v", err)
	}

	return nil
}

func UninstallService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid)
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "stop", "smallweb").Run(); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}

		if err := exec.Command("systemctl", "disable", "smallweb").Run(); err != nil {
			return fmt.Errorf("failed to disable service: %v", err)
		}

		if err := os.Remove(servicePath); err != nil {
			return fmt.Errorf("failed to remove service file: %v", err)
		}

		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			return fmt.Errorf("failed to reload systemd manager configuration: %v", err)
		}

		return nil
	}

	// Stop the service if it is running
	if err := exec.Command("systemctl", "--user", "stop", "smallweb").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	// Disable the service so it doesn't start on boot
	if err := exec.Command("systemctl", "--user", "disable", "smallweb").Run(); err != nil {
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
	uid := os.Getuid()
	if uid == 0 {
		logCmdArg := []string{"smallweb"}
		if follow {
			logCmdArg = append(logCmdArg, "-f")
		}

		logCmd := exec.Command("journalctl", logCmdArg...)
		logCmd.Stdout = os.Stdout
		logCmd.Stderr = os.Stderr
		if err := logCmd.Run(); err != nil {
			return fmt.Errorf("failed to execute journalctl: %v", err)
		}

		return nil
	}

	logCmdArgs := []string{"--user", "--user-unit", "smallweb"}
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
	uid := os.Getuid()

	if uid == 0 {
		statusCmd := exec.Command("systemctl", "status", "smallweb")
		statusCmd.Stdout = os.Stdout
		statusCmd.Stderr = os.Stderr
		if err := statusCmd.Run(); err != nil {
			return fmt.Errorf("failed to get service status: %v", err)
		}
		return nil
	}

	statusCmd := exec.Command("systemctl", "--user", "status", "smallweb")
	statusCmd.Stdout = os.Stdout
	statusCmd.Stderr = os.Stderr
	if err := statusCmd.Run(); err != nil {
		return fmt.Errorf("failed to get service status: %v", err)
	}
	return nil
}
