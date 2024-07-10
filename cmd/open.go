package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/browser"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdOpen() *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "Open the current smallweb app in the browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}

			if dir == worker.SMALLWEB_ROOT {
				return fmt.Errorf("cannot open root directory")
			}

			if !strings.Contains(dir, worker.SMALLWEB_ROOT) {
				return fmt.Errorf("directory is not a smallweb app")
			}

			var hostname string
			if basename := filepath.Base(dir); basename == "@" {
				hostname = filepath.Base(filepath.Dir(dir))
			} else {
				subdomain := filepath.Base(dir)
				domain := filepath.Base(filepath.Dir(dir))
				hostname = fmt.Sprintf("%s.%s", subdomain, domain)
			}

			if err := browser.OpenURL(fmt.Sprintf("https://%s", hostname)); err != nil {
				return fmt.Errorf("failed to open browser: %v", err)
			}

			return nil
		},
	}
}
