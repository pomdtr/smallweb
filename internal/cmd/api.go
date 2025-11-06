package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/pomdtr/smallweb/internal/api"
	"github.com/spf13/cobra"
)

func NewCmdApi() *cobra.Command {
	var flags struct {
		Method string
		Data   string
	}

	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Make API requests to the smallweb API server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiHandler := api.NewHandler(conf)
			if apiHandler == nil {
				return fmt.Errorf("api handler is nil")
			}

			ts := httptest.NewServer(apiHandler)
			defer ts.Close()

			pathname := args[0]
			if !strings.HasPrefix(pathname, "/") {
				pathname = "/" + pathname
			}

			var body io.Reader
			if flags.Data != "" {
				if flags.Data == "@-" {
					body = cmd.InOrStdin()
				} else {
					body = strings.NewReader(flags.Data)
				}
			}

			req, err := http.NewRequest(flags.Method, ts.URL+pathname, body)
			if err != nil {
				return err
			}

			req.Header.Set("X-Forwarded-Host", "api.localhost")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			ct := resp.Header.Get("Content-Type")
			if strings.Contains(strings.ToLower(ct), "json") {
				b, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				var out bytes.Buffer
				if err := json.Indent(&out, b, "", "  "); err != nil {
					// fallback to raw output if indenting fails
					_, err = cmd.OutOrStdout().Write(b)
					return err
				}
				_, err = io.Copy(cmd.OutOrStdout(), &out)
				return err
			}

			_, err = io.Copy(cmd.OutOrStdout(), resp.Body)
			return err
		},
	}

	cmd.Flags().StringVarP(&flags.Method, "method", "X", "GET", "HTTP method to use")
	cmd.Flags().StringVarP(&flags.Data, "data", "d", "", "Data to send in the request body (use @- for stdin)")

	return cmd
}
