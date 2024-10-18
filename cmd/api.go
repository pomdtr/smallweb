package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
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
		Use:     "api",
		Short:   "Interact with the smallweb API",
		GroupID: CoreGroupID,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body io.Reader
			if flags.data != "" {
				body = strings.NewReader(flags.data)
			} else if !isatty.IsTerminal(os.Stdin.Fd()) {
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
