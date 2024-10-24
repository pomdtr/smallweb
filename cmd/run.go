package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pomdtr/smallweb/api"
	"github.com/pomdtr/smallweb/app"
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

			if app.Entrypoint() == "smallweb:api" {
				apiCmd := NewCmdAPI()
				apiCmd.SetArgs(args[1:])

				apiCmd.SetIn(os.Stdin)
				apiCmd.SetOut(os.Stdout)
				apiCmd.SetErr(os.Stderr)

				return apiCmd.Execute()
			}

			if strings.HasPrefix(app.Config.Entrypoint, "smallweb:") {
				return fmt.Errorf("smallweb built-in apps cannot be run")
			}

			worker := worker.NewWorker(app, nil)
			command, err := worker.Command(args[1:]...)
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			return command.Run()
		},
	}

	return cmd
}

func NewCmdAPI() *cobra.Command {
	var flags struct {
		method  string
		headers []string
		data    string
	}

	cmd := &cobra.Command{
		Use:          "api",
		Short:        "Interact with the smallweb API",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body io.Reader
			if flags.data != "" {
				body = strings.NewReader(flags.data)
			} else if flags.data == "@-" {
				body = os.Stdin
			}

			// use api unix socket if available
			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("unix", api.SocketPath(k.String("domain")))
					},
				},
			}

			req, err := http.NewRequest(flags.method, "http://smallweb"+args[0], body)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			for _, header := range flags.headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header: %s", header)
				}

				req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			if resp.Header.Get("Content-Type") == "application/json" {
				var v any
				decoder := json.NewDecoder(resp.Body)
				if err := decoder.Decode(&v); err != nil {
					return fmt.Errorf("failed to decode JSON: %w", err)
				}

				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(v); err != nil {
					return fmt.Errorf("failed to encode JSON: %w", err)
				}

				return nil
			}

			_, _ = io.Copy(os.Stdout, resp.Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.method, "method", "X", "GET", "HTTP method to use")
	cmd.Flags().StringArrayVarP(&flags.headers, "header", "H", nil, "HTTP headers to use")
	cmd.Flags().StringVarP(&flags.data, "data", "d", "", "Data to send in the request body")

	return cmd
}
