package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/app"
	"github.com/spf13/cobra"
)

func NewCmdOpen() *cobra.Command {
	var flags struct {
		app string
	}

	cmd := &cobra.Command{
		Use:   "open [app]",
		Short: "Open an app in the browser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := flags.app
			if flags.app == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("could not get current working directory: %v", err)
				}

				if cwd == path.Clean(k.String("dir")) {
					return fmt.Errorf("not in an app directory")
				}

				if !strings.HasPrefix(cwd, k.String("dir")) {
					return fmt.Errorf("not in an app directory")
				}

				appDir := cwd
				for filepath.Dir(appDir) != k.String("dir") {
					appDir = filepath.Dir(appDir)
				}

				appName = filepath.Base(appDir)
			}

			a, err := app.LoadApp(appName, k.String("dir"), k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			if err := browser.OpenURL(a.URL); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.app, "app", "a", "", "The app to open")
	cmd.RegisterFlagCompletionFunc("app", completeApp)

	return cmd
}
