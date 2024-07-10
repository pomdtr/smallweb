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

type App struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Path string `json:"path"`
}

func ListApps() ([]App, error) {
	domains, err := os.ReadDir(worker.SMALLWEB_ROOT)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
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
			return nil, fmt.Errorf("failed to read directory: %w", err)
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
				Name: fmt.Sprintf("%s.%s", subdomain.Name(), domain.Name()),
				Url:  fmt.Sprintf("https://%s.%s", subdomain.Name(), domain.Name()),
				Path: filepath.Join(worker.SMALLWEB_ROOT, domain.Name(), subdomain.Name()),
			})

		}

	}

	return apps, nil
}

func NewCmdDump() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all smallweb apps",
		GroupID: CoreGroupID,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			apps, err := ListApps()
			if err != nil {
				return fmt.Errorf("failed to list apps: %w", err)
			}

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(apps); err != nil {
				return fmt.Errorf("failed to encode tree: %w", err)
			}

			return nil
		},
	}
	return cmd
}
