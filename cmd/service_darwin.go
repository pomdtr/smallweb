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

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

//go:embed service/com.pomdtr.smallweb.plist
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
			}); err != nil {
				return fmt.Errorf("failed to write service file: %v", err)
			}

			if err := exec.Command("launchctl", "load", servicePath).Run(); err != nil {
				return fmt.Errorf("failed to load service: %v", err)
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

			if flags.follow {
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
		},
	}

	cmd.Flags().BoolVarP(&flags.follow, "follow", "f", false, "Follow log output")
	return cmd
}

func NewCmdServiceStatus() *cobra.Command {
	return &cobra.Command{
		Use: "status",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := exec.Command("launchctl", "list", "com.pomdtr.smallweb").CombinedOutput()
			if err != nil {
				return fmt.Errorf("service is not running")
			}
			fmt.Println("Service is running")
			fmt.Println(string(out))
			return nil
		},
	}
}
