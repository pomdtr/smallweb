package cmd

import (
	"os"
	"os/exec"

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

	return exec.LookPath("deno")
}

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use: "smallweb",
	}

	cmd.AddCommand(NewCmdServe())
	cmd.AddCommand(NewCmdUp())
	cmd.AddCommand(NewCmdAuth())
	cmd.AddCommand(NewCmdProxy())
	cmd.InitDefaultCompletionCmd()

	return cmd
}
