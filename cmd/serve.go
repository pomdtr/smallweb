package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func serializeRequest(req *http.Request) (*Request, error) {
	var res Request

	url := req.URL
	url.Host = req.Host
	if req.TLS != nil {
		url.Scheme = "https"
	} else {
		url.Scheme = "http"
	}
	res.Url = url.String()

	res.Method = req.Method
	for k, v := range req.Header {
		res.Headers = append(res.Headers, []string{k, v[0]})
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	res.Body = body

	return &res, nil
}

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve <app>",
		Short: "Serve a smallweb app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			worker, err := NewWorker(args[0])
			if err != nil {
				return fmt.Errorf("failed to create client: %v", err)
			}

			server := http.Server{
				Addr: fmt.Sprintf(":%d", port),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					req, err := serializeRequest(r)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}

					res, err := worker.Fetch(req)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for _, h := range res.Headers {
						w.Header().Set(h[0], h[1])
					}

					w.WriteHeader(res.Code)
					w.Write(res.Body)
				}),
			}

			fmt.Fprintln(os.Stderr, "Listening on", server.Addr)
			return server.ListenAndServe()
		},
	}
	cmd.Flags().IntP("port", "p", 8000, "Port to listen on")
	return cmd
}
