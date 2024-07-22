package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
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
		Use:     "upgrade [version]",
		Short:   "Upgrade to the latest version",
		GroupID: CoreGroupID,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := cmd.Root().Version
			if version == "dev" {
				fmt.Println("You're compiling from source. Please update manually.")
				return nil
			}

			current, err := semver.NewVersion(version)
			if err != nil {
				return fmt.Errorf("failed to parse current version: %w", err)
			}

			if len(args) > 0 {
				version, err := semver.NewVersion(args[0])
				if err != nil {
					return fmt.Errorf("failed to parse version: %w", err)
				}

				if version.Equal(current) {
					fmt.Printf("version %s is already latest\n", version)
					return nil
				}
				upgradeCmd := fmt.Sprintf("curl -sSfL \"https://install.smallweb.run?version=%s\" | sh", version.String())
				if err := runCommand(upgradeCmd); err != nil {
					return fmt.Errorf("failed to upgrade: %w", err)
				}

				fmt.Println()
				fmt.Println("Ugrade completed successfully")
				fmt.Println("Use `smallweb service restart` to restart the service")
				return nil
			}

			latest, err := fetchLatestVersion()
			if err != nil {
				return fmt.Errorf("failed to get version information: %w", err)
			}

			fmt.Printf("Current version: %s, latest version: %s\n", version, latest)
			if !latest.GreaterThan(current) {
				fmt.Printf("version %s is already latest\n", version)
				return nil
			}

			var upgradeCmd string
			if IsUnderHomebrew() {
				upgradeCmd = "brew update && brew upgrade smallweb"
			} else {
				upgradeCmd = fmt.Sprintf("curl -sSfL \"https://install.smallweb.run?version=%s\" | sh", latest.String())
			}

			if err := runCommand(upgradeCmd); err != nil {
				return fmt.Errorf("failed to upgrade: %w", err)
			}

			fmt.Println("Ugrade completed successfully")
			fmt.Println("Use `smallweb service restart` to restart the service")

			if err := os.Remove(cachedUpgradePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove upgrade cache: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func fetchLatestVersion() (*semver.Version, error) {
	resp, err := http.Get("https://api.smallweb.run/v1/versions/latest")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version information: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read version information: %w", err)
	}

	version, err := semver.NewVersion(strings.TrimSpace(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse version information: %w", err)
	}

	return version, nil
}

func runCommand(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute update command: %w", err)
	}

	return nil
}
