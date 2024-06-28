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
	"time"

	"github.com/pomdtr/smallweb/worker"
)

//go:embed service/com.pomdtr.smallweb.plist
var serviceConfigBytes []byte
var serviceConfig = template.Must(template.New("service").Parse(string(serviceConfigBytes)))

func InstallService() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	servicePath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.pomdtr.smallweb.plist")
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
		"Homedir":  homeDir,
	}); err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	if err := exec.Command("launchctl", "load", servicePath).Run(); err != nil {
		return fmt.Errorf("failed to load service: %v", err)
	}

	return nil
}

func StartService() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	servicePath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.pomdtr.smallweb.plist")
	if !worker.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("launchctl", "start", "com.pomdtr.smallweb").Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func StopService() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	servicePath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.pomdtr.smallweb.plist")
	if !worker.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("launchctl", "stop", "com.pomdtr.smallweb").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	return nil
}

func RestartService() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	servicePath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.pomdtr.smallweb.plist")
	if !worker.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if err := exec.Command("launchctl", "stop", "com.pomdtr.smallweb").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	time.Sleep(2 * time.Second)

	if err := exec.Command("launchctl", "start", "com.pomdtr.smallweb").Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func UninstallService() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	servicePath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.pomdtr.smallweb.plist")
	if !worker.FileExists(servicePath) {
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	servicePath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.pomdtr.smallweb.plist")
	if !worker.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	if !worker.FileExists(servicePath) {
		return fmt.Errorf("service not installed")
	}

	logPath := filepath.Join(homeDir, "Library", "Logs", "smallweb.log")
	if !worker.FileExists(logPath) {
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
	out, err := exec.Command("launchctl", "list", "com.pomdtr.smallweb").CombinedOutput()
	if err != nil {
		return fmt.Errorf("service is not running")
	}
	fmt.Println("Service is running")
	fmt.Println(string(out))
	return nil
}
