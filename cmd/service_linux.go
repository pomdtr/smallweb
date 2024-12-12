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

func getServicePath(uid int, domain string) string {
	if uid == 0 {
		return path.Join("/etc", "systemd", "system", getServiceName(domain)+".service")
	}

	return path.Join(os.Getenv("HOME"), ".config", "systemd", "user", getServiceName(domain)+".service")
}

func InstallService(args []string) error {
	uid := os.Getuid()
	servicePath := getServicePath(uid, k.String("domain"))
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

	username, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	if err := serviceConfig.Execute(f, map[string]any{
		"ExecPath":    execPath,
		"User":        username,
		"SmallwebDir": k.String("dir"),
		"Args":        strings.Join(args, " "),
	}); err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			return fmt.Errorf("failed to reload systemd manager configuration: %v", err)
		}

		if err := exec.Command("systemctl", "enable", getServiceName(k.String("domain"))).Run(); err != nil {
			return fmt.Errorf("failed to enable service: %v", err)
		}

		if err := exec.Command("systemctl", "start", getServiceName(k.String("domain"))).Run(); err != nil {
			return fmt.Errorf("failed to start service: %v", err)
		}

		return nil
	}

	// Reload the systemd manager configuration
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd manager configuration: %v", err)
	}

	// Enable the service to start on boot
	if err := exec.Command("systemctl", "--user", "enable", getServiceName(k.String("domain"))).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %v", err)
	}

	// Start the service immediately
	if err := exec.Command("systemctl", "--user", "start", getServiceName(k.String("domain"))).Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func StartService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid, k.String("domain"))
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "start", getServiceName(k.String("domain"))).Run(); err != nil {
			return fmt.Errorf("failed to start service: %v", err)
		}

		return nil
	}

	if err := exec.Command("systemctl", "--user", "start", getServiceName(k.String("domain"))).Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func StopService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid, k.String("domain"))
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "stop", getServiceName(k.String("domain"))).Run(); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}

		return nil
	}

	if err := exec.Command("systemctl", "--user", "stop", getServiceName(k.String("domain"))).Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	return nil
}

func RestartService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid, k.String("domain"))
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "restart", getServiceName(k.String("domain"))).Run(); err != nil {
			return fmt.Errorf("failed to restart service: %v", err)
		}

		return nil
	}

	if err := exec.Command("systemctl", "--user", "restart", getServiceName(k.String("domain"))).Run(); err != nil {
		return fmt.Errorf("failed to restart service: %v", err)
	}

	return nil
}

func UninstallService() error {
	uid := os.Getuid()
	servicePath := getServicePath(uid, k.String("domain"))
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		if err := exec.Command("systemctl", "stop", getServiceName(k.String("domain"))).Run(); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}

		if err := exec.Command("systemctl", "disable", getServiceName(k.String("domain"))).Run(); err != nil {
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
	if err := exec.Command("systemctl", "--user", "stop", getServiceName(k.String("domain"))).Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	// Disable the service so it doesn't start on boot
	if err := exec.Command("systemctl", "--user", "disable", getServiceName(k.String("domain"))).Run(); err != nil {
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
		return fmt.Errorf("`smallweb service logs` is not supported on Linux")
	}

	servicePath := getServicePath(uid, k.String("domain"))
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if uid == 0 {
		logCmdArg := []string{getServiceName(k.String("domain"))}
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

	logCmdArgs := []string{"--user", "--user-unit", getServiceName(k.String("domain"))}
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
		statusCmd := exec.Command("systemctl", "status", getServiceName(k.String("domain")))
		statusCmd.Stdout = os.Stdout
		statusCmd.Stderr = os.Stderr
		if err := statusCmd.Run(); err != nil {
			return fmt.Errorf("failed to get service status: %v", err)
		}
		return nil
	}

	statusCmd := exec.Command("systemctl", "--user", "status", getServiceName(k.String("domain")))
	statusCmd.Stdout = os.Stdout
	statusCmd.Stderr = os.Stderr
	if err := statusCmd.Run(); err != nil {
		return fmt.Errorf("failed to get service status: %v", err)
	}
	return nil
}
