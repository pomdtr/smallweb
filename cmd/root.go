package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use: "smallweb",
	}

	cmd.AddCommand(NewCmdServe())
	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdList())

	return cmd
}
