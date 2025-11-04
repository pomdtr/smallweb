package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pomdtr/smallweb/internal/utils"
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

			// If the app is not in the config, we're done
			if !slices.Contains(conf.MapKeys("apps"), oldName) {
				fmt.Fprintf(cmd.OutOrStdout(), "Renamed %s to %s\n", oldName, newName)
				return nil
			}

			configPath := utils.FindConfigPath(conf.String("dir"))

			patch := utils.JsonPatch{
				{
					Op:   "move",
					From: fmt.Sprintf("/apps/%s", oldName),
					Path: fmt.Sprintf("/apps/%s", newName),
				},
			}

			if err := utils.PatchFile(configPath, patch); err != nil {
				return fmt.Errorf("updating config file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Renamed %s to %s\n", oldName, newName)
			return nil
		},
	}

	return cmd
}
