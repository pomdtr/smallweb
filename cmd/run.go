package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/auth"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "run <app> [args...]",
		Short:              "Run an app cli",
		GroupID:            CoreGroupID,
		DisableFlagParsing: true,
		SilenceErrors:      true,
		ValidArgsFunction:  completeApp(utils.RootDir()),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}

			rootDir := utils.RootDir()
			app, err := app.LoadApp(filepath.Join(rootDir, args[0]), k.String("domain"))
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}

			if strings.HasPrefix(app.Config.Entrypoint, "smallweb:") {
				return fmt.Errorf("smallweb built-in apps do not support running as a CLI")
			}

			wk := worker.NewWorker(app)
			command, err := wk.Command(args[1:]...)
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if app.Config.Admin {
				token, err := auth.GetApiToken()
				if err != nil {
					return fmt.Errorf("failed to get api token: %w", err)
				}

				freeport, err := worker.GetFreePort()
				if err != nil {
					return fmt.Errorf("failed to get free port: %w", err)
				}

				command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_API_URL=http://127.0.0.1:%d/", freeport))
				command.Env = append(command.Env, fmt.Sprintf("SMALLWEB_API_TOKEN=%s", token))

				go http.ListenAndServe(fmt.Sprintf(":%d", freeport), api.NewHandler(k.String("domain"), nil, nil))
			}

			return command.Run()
		},
	}

	return cmd
}
