package cmd

import (
	"net/url"
	"path/filepath"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "logs",
		Short:             "View app logs",
		Args:              cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: completeApp(),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.LoadApp(filepath.Join(utils.RootDir(), args[0]), k.String("domain"))
			if err != nil {
				return err
			}

			logsUrl, err := url.JoinPath(a.URL, "_logs")
			if err != nil {
				return err
			}

			return browser.OpenURL(logsUrl)
		},
	}

	return cmd
}
