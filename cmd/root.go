package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func denoExecutable() (string, error) {
	if env, ok := os.LookupEnv("DENO_EXEC_PATH"); ok {
		return env, nil
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if denoPath, err := exec.LookPath("deno"); err == nil {
		return denoPath, nil
	}

	denoPath := filepath.Join(homedir, ".deno", "bin", "deno")
	if exists(denoPath) {
		return denoPath, nil
	}

	return "", fmt.Errorf("deno executable not found")
}

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use: "smallweb",
	}

	cmd.AddCommand(NewCmdServe())
	cmd.AddCommand(NewCmdUp())
	cmd.AddCommand(NewCmdAuth())
	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdOpen())
	cmd.AddCommand(NewCmdServer())

	return cmd
}
