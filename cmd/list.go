package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
)

func listApps() ([]string, error) {
	lookupDirs, err := LookupDirs()
	if err != nil {
		return nil, fmt.Errorf("failed to lookup dirs: %v", err)
	}

	apps := make(map[string]struct{})
	for _, lookupDir := range lookupDirs {
		entries, err := os.ReadDir(lookupDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read dir: %v", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			for _, extension := range extensions {
				if exists(path.Join(lookupDir, entry.Name(), "http"+extension)) {
					apps[entry.Name()] = struct{}{}
					break
				}

				if exists(path.Join(lookupDir, "cli", extension)) {
					apps[entry.Name()] = struct{}{}
					break
				}
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
