//go:build darwin

package cmd

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/pomdtr/smallweb/utils"
)

//go:embed embed/com.pomdtr.smallweb.plist
var serviceConfigBytes []byte
var serviceConfig = template.Must(template.New("service").Parse(string(serviceConfigBytes)))
var servicePath = filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.pomdtr.smallweb.plist")

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
		"SmallwebDir": k.String("dir"),
		"ExecPath":    execPath,
		"Args":        args,
		"HomeDir":     os.Getenv("HOME"),
	}); err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	if err := exec.Command("launchctl", "load", servicePath).Run(); err != nil {
		return fmt.Errorf("failed to load service: %v", err)
	}

	return nil
}

func StartService() error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("launchctl", "start", "com.pomdtr.smallweb").Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func StopService() error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("launchctl", "stop", "com.pomdtr.smallweb").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	return nil
}

func RestartService() error {
	return fmt.Errorf("`smallweb service restart` is not supported on macOS")
}

func UninstallService() error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("launchctl", "unload", servicePath).Run(); err != nil {
		return fmt.Errorf("failed to unload service: %v", err)
	}

	if err := os.Remove(servicePath); err != nil {
		return fmt.Errorf("failed to remove service file: %v", err)
	}

	return nil

}

func PrintServiceLogs(follow bool) error {
	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if !utils.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	logPath := filepath.Join(os.Getenv("HOME"), "Library", "Logs", "smallweb.log")
	if !utils.FileExists(logPath) {
		return fmt.Errorf("log file not found")
	}

	if follow {
		cmd := exec.Command("tail", "-f", logPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	f, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open service file: %v", err)
	}

	if _, err := io.Copy(os.Stdout, f); err != nil {
		return fmt.Errorf("failed to copy service file: %v", err)
	}

	return nil

}

func ViewServiceStatus() error {
	cmd := exec.Command("launchctl", "list", "com.pomdtr.smallweb")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute launchctl: %v", err)
	}

	return nil
}
