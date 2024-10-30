package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/atotto/clipboard"
	"github.com/cli/browser"
	"github.com/spf13/cobra"
)

func NewCmdSdk() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "sdk",
		Short:  "Manage smallweb SDK",
		Hidden: true,
	}

	cmd.AddCommand(NewCmdSdkOpen())
	cmd.AddCommand(NewCmdSdkCopy())

	return cmd
}

func NewCmdSdkOpen() *cobra.Command {
	return &cobra.Command{
		Use:  "open",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return browser.OpenURL(args[0])
		},
	}

}

func NewCmdSdkCopy() *cobra.Command {
	return &cobra.Command{
		Use:  "copy",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stdin, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			return clipboard.WriteAll(string(stdin))
		},
	}
}
