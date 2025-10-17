package cmd

import (
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

				relPath, err := filepath.Rel(k.String("dir"), cwd)
				if err != nil {
					cmd.PrintErrln("could not get relative path:", err)
					return ExitError{1}
				}

				parts := strings.Split(relPath, string(os.PathSeparator))
				if len(parts) < 2 {
					cmd.PrintErrln("not in an app directory")
					return ExitError{1}
				}

				appname = parts[1]
			}

			a, err := app.LoadApp(k.String("dir"), k.String("domain"), appname)
			if err != nil {
				cmd.PrintErrf("could not load app %q: %v\n", appname, err)
				return ExitError{1}
			}

			if err := browser.OpenURL(a.URL()); err != nil {
				cmd.PrintErrf("could not open browser for app %q: %v\n", appname, err)
				return ExitError{1}
			}

			return nil
		},
	}

	return cmd
}
