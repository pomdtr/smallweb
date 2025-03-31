package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var appCode = `export default {
    fetch: (_req: Request) => {
        return new Response("Welcome to Smallweb!");
    },
    run: (_args: string[]) => {
        console.log("Welcome to Smallweb!");
    },
};
`

func NewCmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <app>",
		Short: "Create a new app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := k.String("dir")
			appDir := filepath.Join(dir, args[0])
			if _, err := os.Stat(appDir); err == nil {
				cmd.PrintErrf("Directory %s already exists\n", appDir)
				return ErrSilent
			}

			if err := os.MkdirAll(appDir, 0755); err != nil {
				cmd.PrintErrf("Failed to create directory %s: %v\n", appDir, err)
				return ErrSilent
			}

			if err := os.WriteFile(filepath.Join(appDir, "main.ts"), []byte(appCode), 0644); err != nil {
				cmd.PrintErrf("Failed to create main.ts: %v\n", err)
				return ErrSilent
			}

			appDomain := fmt.Sprintf("%s.%s", args[0], k.String("domain"))

			cmd.PrintErrf("App Directory: %s\n", appDir)
			cmd.PrintErrf("App URL: https://%s\n/", appDomain)
			return nil
		},
	}

	return cmd
}
