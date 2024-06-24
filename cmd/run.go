package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/pomdtr/smallweb/client"
	"github.com/spf13/cobra"
)

func installDeno() error {
	// if we are on windows, we need to use the powershell script
	if runtime.GOOS == "windows" {
		return exec.Command("powershell", "-c", "irm https://deno.land/install.ps1 | iex").Run()
	}

	if _, err := exec.LookPath("curl"); err == nil {
		return exec.Command("sh", "-c", "curl -fsSL https://deno.land/x/install/install.sh | sh").Run()
	}

	if _, err := exec.LookPath("wget"); err == nil {
		return exec.Command("sh", "-c", "wget -qO- https://deno.land/x/install/install.sh | sh").Run()
	}

	return nil
}

func setupDenoIfRequired() error {
	if _, err := client.DenoExecutable(); err == nil {
		return nil
	}

	var confirm bool
	form := huh.NewForm(huh.NewGroup(huh.NewConfirm().Description("Deno is required to run smallweb. Do you want to install it now?").Value(&confirm)))
	if err := form.Run(); err != nil {
		return fmt.Errorf("could not get user input: %v", err)
	}

	if !confirm {
		return fmt.Errorf("deno is required to run smallweb")
	}

	fmt.Fprintln(os.Stderr, "Installing deno...")
	if err := installDeno(); err != nil {
		return fmt.Errorf("could not install deno: %v", err)
	}
	fmt.Fprintln(os.Stderr, "Deno installed successfully!")

	return nil
}

func NewCmdRun() *cobra.Command {
	return &cobra.Command{
		Use:                "run <alias> [args...]",
		Short:              "Run a smallweb app cli",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			apps, err := listApps()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			return apps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDenoIfRequired(); err != nil {
				return err
			}

			worker, err := client.NewWorker(args[0])
			if err != nil {
				return err
			}

			return worker.Run(args[1:])
		},
	}

}
