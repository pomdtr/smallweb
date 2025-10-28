package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var template string = `
export default {
  fetch: (_req: Request) => {
	return new Response("Hello, Smallweb!", {
	  headers: { "Content-Type": "text/plain" },
	});
  },
  run: (_args: string[]) => {
	console.log("Hello, Smallweb!");
  },
}
`

func NewCmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [app]",
		Args:  cobra.ExactArgs(1),
		Short: "Create a new Smallweb app",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			appDir := filepath.Join(k.String("dir"), k.String("domain"), args[0])
			if _, err := os.Stat(appDir); !os.IsNotExist(err) {
				cmd.PrintErrf("App directory %s already exists.\n", appDir)
				return ExitError{1}
			}

			if err := os.MkdirAll(appDir, 0755); err != nil {
				return err
			}

			mainFile := filepath.Join(appDir, "main.ts")
			if err := os.WriteFile(mainFile, []byte(template), 0644); err != nil {
				return err
			}

			cmd.Printf("Created new app in %s\n", appDir)
			return nil
		},
	}

	return cmd
}
