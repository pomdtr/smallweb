package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewCmdLink() *cobra.Command {
	var flags struct {
		force bool
	}

	cmd := &cobra.Command{
		Use:     "link <source> <target>",
		Aliases: []string{"ln"},
		Args:    cobra.ExactArgs(2),
		Short:   "Create symbolic links",
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("failed to get absolute path for source: %w", err)
			}

			if _, err := os.Stat(source); err != nil {
				return fmt.Errorf("source does not exist: %w", err)
			}

			if !strings.HasPrefix(source, k.String("dir")) {
				return fmt.Errorf("source must be inside the smallweb directory")
			}

			target, err := filepath.Abs(args[1])
			if err != nil {
				return fmt.Errorf("failed to get absolute path for target: %w", err)
			}

			// if target is inside the smallweb directory, create a relative symlink
			if strings.HasPrefix(target, k.String("dir")) {
				relative, err := filepath.Rel(filepath.Dir(target), source)
				if err != nil {
					return fmt.Errorf("failed to get relative path: %w", err)
				}

				if flags.force {
					if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("failed to remove existing target: %w", err)
					}
				}

				if err := os.Symlink(relative, target); err != nil {
					return fmt.Errorf("failed to create symbolic link: %w", err)
				}

				return nil
			}

			// if target is outside the smallweb directory, create an absolute symlink
			if err := os.Symlink(source, target); err != nil {
				return fmt.Errorf("failed to create symbolic link: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flags.force, "force", "f", false, "Force overwrite existing symlinks")

	return cmd
}
