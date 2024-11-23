package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdDoctor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the system for potential problems 🩺",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "🔍 Checking smallweb directory...")
			if _, err := os.Stat(utils.RootDir()); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "❌ Smallweb directory not found")
				fmt.Fprintln(os.Stderr, "💡 Run `smallweb init` to initialize the workspace")
				return nil
			}
			fmt.Fprintln(os.Stderr, "✅ Smallweb directory found")
			fmt.Fprintln(os.Stderr)

			fmt.Fprintln(os.Stderr, "🔍 Checking Deno version...")
			if err := checkDenoVersion(); err != nil {
				fmt.Fprintln(os.Stderr, "❌ Deno not found")
				fmt.Fprintln(os.Stderr, "💡 Run `curl -fsSL https://deno.land/install.sh | sh` to install Deno")
				return nil
			}
			fmt.Fprintln(os.Stderr, "✅ Deno version is compatible")
			fmt.Fprintln(os.Stderr)

			fmt.Fprintln(os.Stderr, "🎉 smallweb is healthy")
			return nil
		},
	}

	return cmd
}

func checkDenoVersion() error {
	deno, err := exec.LookPath("deno")
	if err != nil {
		return err
	}

	cmd := exec.Command(deno, "eval", "--print", "Deno.version.deno")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	denoVersion := strings.Trim(string(out), "\n")
	v, err := semver.NewVersion(denoVersion)
	if err != nil {
		return err
	}

	if v.Major() < 2 {
		fmt.Fprintln(os.Stderr, "Deno version 2 or higher is required")
		fmt.Fprintln(os.Stderr, "Run `deno upgrade` to upgrade Deno")
		return nil
	}

	return nil
}
