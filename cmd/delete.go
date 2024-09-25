package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete an app",
		Aliases:           []string{"remove", "rm"},
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		GroupID:           CoreGroupID,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			p := filepath.Join(rootDir, args[0])
			if _, err := os.Stat(p); os.IsNotExist(err) {
				return fmt.Errorf("app not found: %s", args[0])
			}

			if err := os.RemoveAll(p); err != nil {
				return fmt.Errorf("failed to delete app: %w", err)
			}

			cmd.Printf("App %s deleted\n", args[0])
			return nil
		},
	}

	return cmd
}
