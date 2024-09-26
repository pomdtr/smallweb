package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/api"
	"github.com/spf13/cobra"
)

func NewCmdAPI() *cobra.Command {
	var flags struct {
		method  string
		headers []string
		data    string
	}

	cmd := &cobra.Command{
		Use:   "api",
		Short: "Interact with the smallweb API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := api.NewHandler(k)

			var body io.Reader
			if flags.data != "" {
				body = strings.NewReader(flags.data)
			} else if !isatty.IsTerminal(os.Stdin.Fd()) {
				body = os.Stdin
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(flags.method, args[0], body)
			for _, header := range flags.headers {
				parts := strings.SplitN(header, ":", 2)
				req.Header.Add(parts[0], parts[1])
			}

			handler.ServeHTTP(w, req)
			resp := w.Result()

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
