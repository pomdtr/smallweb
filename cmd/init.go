package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

//go:embed embed/workspace
var embedFS embed.FS

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <domain>",
		Short: "Initialize a new workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceFS, err := fs.Sub(embedFS, "embed/workspace")
			if err != nil {
				return fmt.Errorf("failed to read workspace embed: %w", err)
			}

			workspaceDir := k.String("dir")
			if _, err := fs.Stat(workspaceFS, workspaceDir); err == nil {
				return fmt.Errorf("directory %s already exists", workspaceDir)
			}

			if err := os.CopyFS(workspaceDir, workspaceFS); err != nil {
				return fmt.Errorf("failed to copy workspace embed: %w", err)
			}

			configPath := path.Join(workspaceDir, ".smallweb", "config.json")
			if err := os.MkdirAll(path.Dir(configPath), 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			configFile, err := os.Create(configPath)
			if err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}

			encoder := json.NewEncoder(configFile)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(map[string]interface{}{
				"domain": args[0],
				"adminApps": []string{
					"vscode",
				},
			}); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			if err := godotenv.Write(map[string]string{
				"VSCODE_PASSWORD": uuid.NewString(),
			}, path.Join(workspaceDir, "vscode", ".env")); err != nil {
				return fmt.Errorf("failed to write .env file: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Workspace initialized at %s\n", workspaceDir)
			return nil
		},
	}

	return cmd
}
