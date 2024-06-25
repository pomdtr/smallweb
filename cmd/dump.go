package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

type Tree struct {
	Root string         `json:"root"`
	Apps map[string]App `json:"apps"`
}

type App struct {
	Cli  string `json:"cli,omitempty"`
	Name string `json:"name"`
	Main string `json:"main,omitempty"`
}

func NewCmdDump() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Print the smallweb app tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := os.ReadDir(worker.SMALLWEB_ROOT)
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

				app := App{
					Name: entry.Name(),
				}

				for _, extension := range worker.EXTENSIONS {
					if worker.FileExists(worker.SMALLWEB_ROOT, entry.Name(), "cli"+extension) {
						app.Cli = "cli" + extension
					}

					if worker.FileExists(worker.SMALLWEB_ROOT, entry.Name(), "main"+extension) {
						app.Main = "main" + extension
					}

				}

				apps[entry.Name()] = app
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
