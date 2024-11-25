package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

//go:embed embed/smallweb
var initFS embed.FS

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize smallweb workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.RootDir()
			if _, err := os.Stat(rootDir); !os.IsNotExist(err) {
				return fmt.Errorf("smallweb directory already exists: %s", rootDir)
			}

			if err := os.CopyFS(rootDir, initFS); err != nil {
				return fmt.Errorf("failed to copy template: %w", err)
			}

			fmt.Printf("Smallweb dir initialized at %s\n", rootDir)
			return nil
		},
	}

	return cmd
}
