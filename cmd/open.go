package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open [app]",
		Short: "Open the smallweb app specified by dir in the browser",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
		},
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := utils.ExpandTilde(k.String("dir"))
			if len(args) > 0 {
				for _, app := range ListApps(rootDir) {
					if app != args[0] {
						continue
					}

					if cnamePath := filepath.Join(rootDir, app, "CNAME"); utils.FileExists(cnamePath) {
						cname, err := os.ReadFile(cnamePath)
						if err != nil {
							return fmt.Errorf("failed to read CNAME file: %v", err)
						}

						url := fmt.Sprintf("https://%s/", string(cname))
						if err := browser.OpenURL(url); err != nil {
							return fmt.Errorf("failed to open browser: %v", err)
						}

						return nil
					}

					url := fmt.Sprintf("https://%s.%s/", app, k.String("domain"))
					if err := browser.OpenURL(url); err != nil {
						return fmt.Errorf("failed to open browser: %v", err)
					}
					return nil
				}

				return fmt.Errorf("app not found: %s", args[0])
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current dir: %v", err)
			}

			if filepath.Dir(cwd) != rootDir {
				return fmt.Errorf("current dir is not a smallweb app")
			}

			if cnamePath := filepath.Join(cwd, "CNAME"); utils.FileExists(cnamePath) {
				cname, err := os.ReadFile(cnamePath)
				if err != nil {
					return fmt.Errorf("failed to read CNAME file: %v", err)
				}

				url := fmt.Sprintf("https://%s/", string(cname))
				if err := browser.OpenURL(url); err != nil {
					return fmt.Errorf("failed to open browser: %v", err)
				}

				return nil
			}

			url := fmt.Sprintf("https://%s.%s/", filepath.Base(cwd), k.String("domain"))
			if err := browser.OpenURL(url); err != nil {
				return fmt.Errorf("failed to open browser: %v", err)
			}

			return nil
		},
	}

	return cmd
}
