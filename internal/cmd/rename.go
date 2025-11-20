package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewCmdRename() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "rename <old-name> <new-name>",
		Aliases:           []string{"mv"},
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeApp,
		Short:             "Rename a file or directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			oldPath := filepath.Join(conf.String("dir"), oldName)
			newPath := filepath.Join(conf.String("dir"), newName)

			// Check if old path exists
			if _, err := os.Stat(oldPath); os.IsNotExist(err) {
				cmd.PrintErrf("path %q does not exist\n", oldName)
				return ExitError{1}
			}

			// Check if new path already exists
			if _, err := os.Stat(newPath); err == nil {
				cmd.PrintErrf("path %q already exists\n", newName)
				return ExitError{1}
			}

			// Rename the directory
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to rename %q to %q: %w", oldName, newName, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Renamed %s to %s\n", oldName, newName)
			return nil
		},
	}

	return cmd
}
