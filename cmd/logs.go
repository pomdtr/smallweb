package cmd

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdLogs() *cobra.Command {
	var flags struct {
		http    bool
		console bool
	}

	cmd := &cobra.Command{
		Use:               "log",
		Aliases:           []string{"logs"},
		Short:             "View app logs",
		Args:              cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: completeApp(),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.LoadApp(filepath.Join(utils.RootDir(), args[0]), k.String("domain"))
			if err != nil {
				return err
			}

			var logPath string
			if flags.console {
				logPath = "_logs/console"
			} else if flags.http {
				logPath = "_logs/http"
			} else {
				return fmt.Errorf("one of --http or --console is required")
			}

			logsUrl, err := url.JoinPath(a.URL, logPath)
			if err != nil {
				return err
			}

			return browser.OpenURL(logsUrl)
		},
	}

	cmd.Flags().BoolVar(&flags.http, "http", false, "View logs in the browser")
	cmd.Flags().BoolVar(&flags.console, "console", false, "View logs in the console")
	cmd.MarkFlagsOneRequired("http", "console")

	return cmd
}
