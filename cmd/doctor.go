package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"
)

var minimumDenoVersion = semver.MustParse("2.2.0")

func NewCmdDoctor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the system for potential problems",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "🔍 Checking smallweb directory...")
			if _, err := os.Stat(k.String("dir")); os.IsNotExist(err) {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Smallweb directory not found")
				fmt.Fprintln(cmd.ErrOrStderr(), "💡 Run `smallweb init` to initialize the workspace")
				return nil
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "✅ Smallweb directory found")
			fmt.Fprintln(cmd.ErrOrStderr())

			fmt.Fprintln(cmd.ErrOrStderr(), "🔍 Checking Deno version...")
			version, err := checkDenoVersion()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Deno not found")
				fmt.Fprintln(cmd.ErrOrStderr(), "💡 Run `curl -fsSL https://deno.land/install.sh | sh` to install Deno")
				return nil
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "✅ Deno version is compatible (%s)\n", version)
			fmt.Fprintln(cmd.ErrOrStderr())

			fmt.Fprintln(cmd.ErrOrStderr(), "🔍 Checking domain...")
			if k.String("domain") == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Domain not set")
				fmt.Fprintf(cmd.ErrOrStderr(), "💡 Set it using the $SMALLWEB_DOMAIN env var or the `domain` field in your smallweb config")
				return nil
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "✅ Domain is set")
			fmt.Fprintln(cmd.ErrOrStderr())

			fmt.Fprintln(cmd.ErrOrStderr(), "🎉 smallweb is healthy")
			return nil
		},
	}

	return cmd
}

func checkDenoVersion() (string, error) {
	deno, err := exec.LookPath("deno")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(deno, "eval", "--print", "Deno.version.deno")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	denoVersion := strings.Trim(string(out), "\n")
	v, err := semver.NewVersion(denoVersion)
	if err != nil {
		return "", err
	}

	if v.LessThan(minimumDenoVersion) {
		return denoVersion, fmt.Errorf("deno version %s is too old, please upgrade to %s or newer", denoVersion, minimumDenoVersion)
	}

	return v.String(), nil
}
