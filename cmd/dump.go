package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

type Tree struct {
	Root string `json:"root"`
	Apps []App  `json:"apps"`
}

type App struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Root string `json:"root"`
}

func NewCmdDump() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dump",
		Short:   "Print the smallweb app tree",
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			domains, err := os.ReadDir(worker.SMALLWEB_ROOT)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			apps := make([]App, 0)
			for _, domain := range domains {
				if !domain.IsDir() {
					continue
				}

				// Skip hidden files
				if strings.HasPrefix(domain.Name(), ".") {
					continue
				}

				subdomain, err := os.ReadDir(filepath.Join(worker.SMALLWEB_ROOT, domain.Name()))
				if err != nil {
					return fmt.Errorf("failed to read directory: %w", err)
				}

				for _, subdomain := range subdomain {
					if !subdomain.IsDir() {
						continue
					}

					// Skip hidden files
					if strings.HasPrefix(subdomain.Name(), ".") {
						continue
					}

					apps = append(apps, App{
						Name: subdomain.Name(),
						Url:  fmt.Sprintf("https://%s.%s", subdomain.Name(), domain.Name()),
						Root: filepath.Join(worker.SMALLWEB_ROOT, domain.Name(), subdomain.Name()),
					})

				}

			}

			tree := Tree{
				Root: worker.SMALLWEB_ROOT,
				Apps: apps,
			}

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(tree); err != nil {
				return fmt.Errorf("failed to encode tree: %w", err)
			}

			return nil
		},
	}
	return cmd
}
