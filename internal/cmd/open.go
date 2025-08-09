package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "open [app]",
		Short:             "Open an app in the browser",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeApp,
		RunE: func(cmd *cobra.Command, args []string) error {
			var appname string
			if len(args) > 0 {
				appname = args[0]
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					cmd.PrintErrln("could not get current working directory:", err)
					return ExitError{1}
				}

				if cwd == path.Clean(k.String("dir")) {
					cmd.PrintErrln("not in an app directory")
					return ExitError{1}
				}

				if !strings.HasPrefix(cwd, k.String("dir")) {
					cmd.PrintErrln("not in an app directory")
					return ExitError{1}
				}

				appDir := cwd
				for filepath.Dir(appDir) != k.String("dir") {
					appDir = filepath.Dir(appDir)
				}

				appname = filepath.Base(appDir)
			}

			a, err := app.LoadApp(appname)
			if err != nil {
				cmd.PrintErrf("could not load app %q: %v\n", appname, err)
				return ExitError{1}
			}

			if err := browser.OpenURL(fmt.Sprintf("http://%s.%s", a.Name, k.String("domain"))); err != nil {
				cmd.PrintErrf("could not open browser for app %q: %v\n", appname, err)
				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}
