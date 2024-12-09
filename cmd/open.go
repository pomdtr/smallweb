package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "open [app]",
		Short:             "Open an app in the browser",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeApp(utils.RootDir),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.RootDir

			if len(args) == 0 {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				if filepath.Dir(cwd) != rootDir {
					return fmt.Errorf("no app specified and not in an app directory")
				}

				a, err := app.NewApp(filepath.Base(cwd), rootDir, k.String("domain"), slices.Contains(k.Strings("adminApps"), filepath.Base(cwd)))
				if err != nil {
					return fmt.Errorf("failed to load app: %w", err)
				}

				if err := browser.OpenURL(a.URL); err != nil {
					return fmt.Errorf("failed to open browser: %w", err)
				}

				return nil
			}

			a, err := app.NewApp(args[0], rootDir, k.String("domain"), slices.Contains(k.Strings("adminApps"), args[0]))
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			if err := browser.OpenURL(a.URL); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			return nil
		},
	}

	return cmd
}
