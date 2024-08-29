package cmd

import (
	"fmt"
	"os"

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
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			return completeApp(cmd, args, toComplete)
		},
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("dir")))
				if err != nil {
					return fmt.Errorf("failed to list apps: %v", err)
				}

				for _, app := range apps {
					if app.Name != args[0] {
						continue
					}

					url := fmt.Sprintf("https://%s/", app.Hostname)
					if err := browser.OpenURL(url); err != nil {
						return fmt.Errorf("failed to open browser: %v", err)
					}
					return nil
				}

				return fmt.Errorf("app not found: %s", args[0])
			}

			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current dir: %v", err)
			}

			apps, err := ListApps(k.String("domain"), utils.ExpandTilde(k.String("dir")))
			for _, app := range apps {
				if app.Dir != dir {
					continue
				}

				url := fmt.Sprintf("https://%s/", app.Hostname)
				if err := browser.OpenURL(url); err != nil {
					return fmt.Errorf("failed to open browser: %v", err)
				}
			}

			return fmt.Errorf("current dir is not a smallweb app")
		},
	}

	return cmd
}
