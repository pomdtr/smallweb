package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <template> [app]",
		Aliases: []string{"new"},
		Short:   "Create a new smallweb app",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			template := args[0]

			var name string
			if len(args) == 2 {
				name = args[1]
			} else {
				parts := strings.Split(template, "/")
				name = parts[len(parts)-1]
				name = strings.TrimSuffix(name, "-template")
			}

			appDir := filepath.Join(utils.RootDir, name)
			if _, err := os.Stat(appDir); !os.IsNotExist(err) {
				return fmt.Errorf("directory already exists: %s", appDir)
			}

			if err := getter.Get(appDir, template); err != nil {
				return fmt.Errorf("failed to get template: %w", err)
			}

			cmd.Printf("App initialized, you can now access it at https://%s.%s\n", name, k.String("domain"))
			return nil
		},
	}

	return cmd
}
