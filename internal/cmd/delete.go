package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pomdtr/smallweb/internal/utils"
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
			appDir := filepath.Join(k.String("dir"), args[0])
			if _, err := os.Stat(appDir); os.IsNotExist(err) {
				cmd.PrintErrf("path %q does not exist\n", args[0])
				return ExitError{1}
			}

			if err := os.RemoveAll(appDir); err != nil {
				return fmt.Errorf("failed to delete %q: %w", args[0], err)
			}

			if !slices.Contains(k.MapKeys("apps"), args[0]) {
				fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s\n", args[0])
				return nil
			}

			jsonPath := fmt.Sprintf("/%s/%s", "apps", args[0])
			patch := utils.JsonPatch{
				{
					Op:   "remove",
					Path: jsonPath,
				},
			}

			configPath := utils.FindConfigPath(k.String("dir"))
			if err := utils.PatchFile(configPath, patch); err != nil {
				return fmt.Errorf("updating config file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Deleted", args[0])
			return nil
		},
	}

	return cmd
}
