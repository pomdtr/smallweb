package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
			var method string
			var url string
			headers := make(http.Header)
			var body io.Reader

			if args[0] == "-" {
				if isatty.IsTerminal(os.Stdin.Fd()) {
					return fmt.Errorf("no input data provided")
				}

				inputBytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}

				payload := struct {
					URL     string            `json:"url"`
					Method  string            `json:"method"`
					Headers map[string]string `json:"headers"`
					Body    []byte            `json:"body"`
				}{
					Method:  "GET",
					Headers: make(map[string]string),
				}

				if err := json.Unmarshal(inputBytes, &payload); err != nil {
					return fmt.Errorf("failed to unmarshal input: %w", err)
				}

				if payload.URL == "" {
					return fmt.Errorf("url cannot be empty")
				}

				method = payload.Method
				url = payload.URL
				for k, v := range payload.Headers {
					headers.Add(k, v)
				}

				if payload.Body != nil {
					body = bytes.NewReader(payload.Body)
				}

			} else {
				method = flags.method
				url = fmt.Sprintf("http://api" + args[0])
				for _, header := range flags.headers {
					parts := strings.SplitN(header, ":", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid header: %s", header)
					}

					headers.Add(parts[0], parts[1])
				}

				if flags.data != "" {
					body = strings.NewReader(flags.data)
				} else if flags.data == "@-" {
					body = os.Stdin
				}
			}

			req := httptest.NewRequest(method, url, body)
			req.Header = headers

			handler := api.NewHandler(k.String("domain"))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if args[0] != "-" {
				io.Copy(os.Stdout, w.Body)
				return nil
			}

			b, err := io.ReadAll(w.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			res := struct {
				Status     int               `json:"status"`
				StatusText string            `json:"statusText"`
				Headers    map[string]string `json:"headers"`
				Body       []byte            `json:"body"`
			}{
				Status:     w.Code,
				StatusText: http.StatusText(w.Code),
				Headers:    make(map[string]string),
				Body:       b,
			}

			for k, v := range w.Header() {
				res.Headers[k] = v[0]
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(res); err != nil {
				return fmt.Errorf("failed to encode response: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.method, "method", "X", "GET", "HTTP method to use")
	cmd.Flags().StringArrayVarP(&flags.headers, "header", "H", nil, "HTTP headers to use")
	cmd.Flags().StringVarP(&flags.data, "data", "d", "", "Data to send in the request body")

	return cmd
}
