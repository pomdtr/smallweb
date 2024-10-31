package cmd

import (
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/net/webdav"
)

func NewCmdWebdav() *cobra.Command {
	var flags struct {
		method  string
		headers []string
		data    string
	}

	cmd := &cobra.Command{
		Use:   "webdav",
		Short: "Interact with the smallweb WebDAV server",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var handler = &webdav.Handler{
				FileSystem: webdav.Dir(utils.RootDir()),
				LockSystem: webdav.NewMemLS(),
			}
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

			u, err := url.JoinPath("http://webdav/", args[0])
			if err != nil {
				return fmt.Errorf("path is not valid: %w", err)
			}
			req := httptest.NewRequest(flags.method, u, body)
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