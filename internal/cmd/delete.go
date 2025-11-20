package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete <path>",
		Aliases:           []string{"del", "rm"},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp,
		Short:             "Delete a file or directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(conf.String("dir"), args[0])
			if _, err := os.Stat(appDir); os.IsNotExist(err) {
				cmd.PrintErrf("path %q does not exist\n", args[0])
				return ExitError{1}
			}

			if err := os.RemoveAll(appDir); err != nil {
				return fmt.Errorf("failed to delete %q: %w", args[0], err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Deleted", args[0])
			return nil
		},
	}

	return cmd
}
