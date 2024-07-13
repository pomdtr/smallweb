package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func IsUnderHomebrew() bool {
	binary, err := os.Executable()
	if err != nil {
		return false
	}

	brewExe, err := exec.LookPath("brew")
	if err != nil {
		return false
	}

	brewPrefixBytes, err := exec.Command(brewExe, "--prefix").Output()
	if err != nil {
		return false
	}

	brewBinPrefix := filepath.Join(strings.TrimSpace(string(brewPrefixBytes)), "bin") + string(filepath.Separator)
	return strings.HasPrefix(binary, brewBinPrefix)
}

func NewCmdUpgrade() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade to the latest version",
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			latest, err := fetchLatestVersion()
			if err != nil {
				return fmt.Errorf("failed to get version information: %w", err)
			}

			version := cmd.Root().Version
			fmt.Printf("Current version: %s, latest version: %s\n", version, latest)
			if version == "dev" {
				fmt.Println("You're compiling from source. Please update manually.")
				return nil
			} else if version >= latest {
				fmt.Printf("version %s is already latest\n", version)
				return nil
			}

			return Update()
		},
	}

	return cmd
}

func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://assets.smallweb.run/version.txt")
	if err != nil {
		return "", fmt.Errorf("failed to fetch version information: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read version information: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}

func Update() error {
	var updateCmd string

	if IsUnderHomebrew() {
		updateCmd = "brew update && brew upgrade smallweb"
	} else {
		updateCmd = "curl -sSfL \"https://install.smallweb.run\" | sh"
	}
	command := exec.Command("sh", "-c", updateCmd)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err != nil {
		return fmt.Errorf("failed to execute update command: %w", err)
	}

	fmt.Println("Update completed successfully")
	fmt.Println("Use `smallweb service restart` to restart the service")
	return nil
}
