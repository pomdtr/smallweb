package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := api.NewHandler(k.String("domain"))
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

			url, err := url.JoinPath("http://api/", args[0])
			if err != nil {
				return fmt.Errorf("path is not valid: %w", err)
			}
			req := httptest.NewRequest(flags.method, url, body)
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

func serveFromStream(handler http.Handler, input io.Reader, outpput io.Writer) error {
	inputBytes, err := io.ReadAll(input)
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

	req := httptest.NewRequest(payload.Method, payload.URL, bytes.NewReader(payload.Body))
	for k, v := range payload.Headers {
		req.Header.Add(k, v)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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

	encoder := json.NewEncoder(outpput)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(res); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

type HttpClient struct {
	handler http.Handler
}

func (c *HttpClient) Do(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func NewApiClient(domain string) (*api.ClientWithResponses, error) {
	handler := api.NewHandler(domain)
	return api.NewClientWithResponses("", api.WithHTTPClient(&HttpClient{handler: handler}))
}
