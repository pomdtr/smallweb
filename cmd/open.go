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
		Short: "Open an app in the browser",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			rootDir := utils.ExpandTilde(k.String("dir"))
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			return ListApps(rootDir), cobra.ShellCompDirectiveNoFileComp
		},
		GroupID: CoreGroupID,
		Run: func(cmd *cobra.Command, args []string) {
			rootDir := utils.ExpandTilde(k.String("dir"))

			if args[0] == "cli" {
				if err := browser.OpenURL(fmt.Sprintf("https://cli.%s/", k.String("domain"))); err != nil {
					cmd.PrintErrf("failed to open browser: %v", err)
					os.Exit(1)
				}
			}

			for _, app := range ListApps(rootDir) {
				if app != args[0] {
					continue
				}

				if cnamePath := filepath.Join(rootDir, app, "CNAME"); utils.FileExists(cnamePath) {
					cname, err := os.ReadFile(cnamePath)
					if err != nil {
						cmd.PrintErrf("failed to read CNAME file: %v", err)
						os.Exit(1)
					}

					url := fmt.Sprintf("https://%s/", string(cname))
					if err := browser.OpenURL(url); err != nil {
						cmd.PrintErrf("failed to open browser: %v", err)
						os.Exit(1)
					}
				}

				url := fmt.Sprintf("https://%s.%s/", app, k.String("domain"))
				if err := browser.OpenURL(url); err != nil {
					cmd.PrintErrf("failed to open browser: %v", err)
					os.Exit(1)
				}
				return
			}

			cmd.PrintErrf("app not found: %s", args[0])
			os.Exit(1)
		},
	}

	return cmd
}
