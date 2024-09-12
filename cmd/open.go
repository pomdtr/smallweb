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
		Use:               "open [app]",
		Short:             "Open an app in the browser",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp(utils.ExpandTilde(k.String("dir"))),
		GroupID:           CoreGroupID,
		Run: func(cmd *cobra.Command, args []string) {
			url := fmt.Sprintf("https://%s.%s/", args[0], k.String("domain"))
			if err := browser.OpenURL(url); err != nil {
				cmd.PrintErrf("failed to open browser: %v", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}
