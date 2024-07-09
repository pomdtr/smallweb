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
	Root string         `json:"root"`
	Apps map[string]App `json:"apps"`
}

type App struct {
	Name string `json:"name"`
	Root string `json:"root"`
}

func NewCmdDump() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dump <base>",
		Short:   "Print the smallweb app tree",
		Args:    cobra.ExactArgs(1),
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := filepath.Join(worker.SMALLWEB_ROOT, args[0])
			entries, err := os.ReadDir(rootDir)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			apps := make(map[string]App)
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				// Skip hidden files
				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				apps[entry.Name()] = App{
					Name: entry.Name(),
					Root: filepath.Join(rootDir, entry.Name()),
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
