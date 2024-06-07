package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/pomdtr/smallweb/client"
	"github.com/spf13/cobra"
)

func listApps() ([]string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home dir: %v", err)
	}

	apps := make(map[string]struct{})
	rootDir := path.Join(homedir, "www")
	entries, err := os.ReadDir(path.Join(rootDir))
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		for _, extension := range client.EXTENSIONS {
			if exists(path.Join(rootDir, entry.Name(), "http"+extension)) {
				apps[entry.Name()] = struct{}{}
				break
			}

			if exists(path.Join(rootDir, "cli", extension)) {
				apps[entry.Name()] = struct{}{}
				break
			}

			if exists(path.Join(rootDir, entry.Name(), "index.html")) {
				apps[entry.Name()] = struct{}{}
				break
			}
		}
	}

	var appList []string
	for app := range apps {
		appList = append(appList, app)
	}

	return appList, nil
}

func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			apps, err := listApps()
			if err != nil {
				return err
			}

			for _, app := range apps {
				cmd.Println(app)
			}

			return nil
		},
	}
}
