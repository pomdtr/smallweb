package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"

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
			server := api.NewServer(k)
			handler := api.Handler(&server)

			var body io.Reader
			if flags.data != "" {
				if flags.method == "GET" {
					return fmt.Errorf("cannot send data with GET request")
				}

				if flags.data == "@-" {
					body = os.Stdin
				} else if flags.data[0] == '@' {
					file, err := os.Open(flags.data[1:])
					if err != nil {
						return fmt.Errorf("failed to open file: %w", err)
					}
					defer file.Close()

					body = file
				} else {
					body = strings.NewReader(flags.data)
				}
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
