package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"
)

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
				fmt.Println("Ugrade completed successfully!")
				fmt.Println("Make sure to restart the smallweb service.")
				return nil
			}

			var upgradeCmd string
			if IsUnderHomebrew() {
				upgradeCmd = "brew update && brew upgrade smallweb"
			} else {
				upgradeCmd = "curl -sSfL \"https://install.smallweb.run\" | sh"
			}

			if err := runCommand(upgradeCmd); err != nil {
				return fmt.Errorf("failed to upgrade: %w", err)
			}

			fmt.Println("Ugrade completed successfully")
			fmt.Println("Make sure to restart smallweb to apply the changes")
			return nil
		},
	}

	return cmd
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
