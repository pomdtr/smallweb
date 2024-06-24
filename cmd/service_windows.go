//go:build windows

package cmd

import "fmt"

func InstallService() error {
	return fmt.Errorf("service installation is not supported on Windows")
}

func UninstallService() error {
	return fmt.Errorf("service uninstallation is not supported on Windows")
}

func PrintServiceLogs(_ bool) error {
	return fmt.Errorf("service status is not supported on Windows")
}

func ViewServiceStatus() error {
	return fmt.Errorf("service log is not supported on Windows")
}
