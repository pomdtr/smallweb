package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func NewCmdFetch() *cobra.Command {
	var flags struct {
		method  string
		headers []string
		data    string
	}

	cmd := &cobra.Command{
		Use:               "fetch <app> <path>",
		Short:             "Fetch a path from an app",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp(),
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ServeApp(w, r, true)
			})
			if len(args) == 0 {
				if isatty.IsTerminal(os.Stdin.Fd()) {
					return cmd.Help()
				}

				return serveFromStream(handler, os.Stdin, os.Stdout)
			}

			var body io.Reader
			if flags.data != "" {
				body = strings.NewReader(flags.data)
			} else if flags.data == "@-" {
				body = os.Stdin
			}

			req := httptest.NewRequest(flags.method, args[0], body)
			for _, header := range flags.headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header: %s", header)
				}

				req.Header.Add(parts[0], parts[1])
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			io.Copy(os.Stdout, w.Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.method, "method", "X", "GET", "HTTP method to use")
	cmd.Flags().StringArrayVarP(&flags.headers, "header", "H", nil, "HTTP headers to use")
	cmd.Flags().StringVarP(&flags.data, "data", "d", "", "Data to send in the request body")

	return cmd
}
