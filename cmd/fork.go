package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdFork() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "fork [app] [new-name]",
		Short:             "Fork an app",
		Aliases:           []string{"cp"},
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		Args:              cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			src := filepath.Join(rootDir, args[0])
			dst := filepath.Join(rootDir, args[1])

			if _, err := os.Stat(src); os.IsNotExist(err) {
				return fmt.Errorf("app not found: %s", args[0])
			}

			if _, err := os.Stat(dst); !os.IsNotExist(err) {
				return fmt.Errorf("app already exists: %s", args[1])
			}

			fs := os.DirFS(src)
			if err := os.CopyFS(dst, fs); err != nil {
				return fmt.Errorf("failed to copy app: %w", err)
			}

			cmd.Printf("App %s renamed to %s\n", args[0], args[1])
			return nil
		},
	}

	return cmd
}
