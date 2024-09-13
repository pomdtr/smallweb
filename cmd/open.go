package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "open [app]",
		Short:             "Open an app in the browser",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		GroupID:           CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			a, err := app.LoadApp(filepath.Join(rootDir, args[0]), k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			if err := browser.OpenURL(a.Url); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			return nil
		},
	}

	return cmd
}
