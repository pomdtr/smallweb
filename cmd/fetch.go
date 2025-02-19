package cmd

import (
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/worker"
	"github.com/spf13/cobra"
)

func NewCmdFetch() *cobra.Command {
	var flags struct {
		method  string
		headers []string
		data    string
	}

	cmd := &cobra.Command{
		Use:               "fetch <app> [path]",
		Short:             "Fetch a path from an app",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: completeApp,
		RunE: func(cmd *cobra.Command, args []string) error {
			var body io.Reader
			if flags.data != "" {
				body = strings.NewReader(flags.data)
			} else if flags.data == "@-" {
				body = os.Stdin
			}

			var requestPath string
			if len(args) == 2 {
				requestPath = args[1]
			} else {
				requestPath = "/"
			}

			req := httptest.NewRequest(flags.method, requestPath, body)
			for _, header := range flags.headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header: %s", header)
				}

				req.Header.Add(parts[0], parts[1])
			}

			req.Host = fmt.Sprintf("%s.%s", args[0], k.String("domain"))

			a, err := app.LoadApp(args[0], k.String("dir"), k.String("domain"), k.Bool(fmt.Sprintf("apps.%s.admin", args[0])))
			if err != nil {
				return fmt.Errorf("failed to load app: %w", err)
			}

			wk := worker.NewWorker(a)
			if err := wk.Start(); err != nil {
				return fmt.Errorf("failed to start worker: %w", err)
			}

			defer wk.Stop()

			w := httptest.NewRecorder()
			wk.ServeHTTP(w, req)
			_, _ = io.Copy(cmd.OutOrStdout(), w.Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.method, "method", "X", "GET", "HTTP method to use")
	cmd.Flags().StringArrayVarP(&flags.headers, "header", "H", nil, "HTTP headers to use")
	cmd.Flags().StringVarP(&flags.data, "data", "d", "", "Data to send in the request body. Use @- to read from stdin")

	return cmd
}
