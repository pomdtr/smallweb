package cmd

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdTunnel() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "tunnel",
		Short:             "Start a tunnel to a remote server (powered by localhost.run)",
		GroupID:           CoreGroupID,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp(utils.RootDir()),
		RunE: func(cmd *cobra.Command, args []string) error {
			server := http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					req, err := http.NewRequest(r.Method, fmt.Sprintf("http://%s:%d%s", k.String("host"), k.Int("port"), r.URL.Path), r.Body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for k, v := range r.Header {
						for _, vv := range v {
							req.Header.Add(k, vv)
						}
					}
					req.Header.Add("X-Forwarded-Host", fmt.Sprintf("%s.%s", args[0], k.String("domain")))

					client := &http.Client{
						CheckRedirect: func(req *http.Request, via []*http.Request) error {
							return http.ErrUseLastResponse
						},
					}

					resp, err := client.Do(req)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for k, v := range resp.Header {
						for _, vv := range v {
							w.Header().Add(k, vv)
						}
					}

					w.WriteHeader(resp.StatusCode)
					_, err = io.Copy(w, resp.Body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}),
			}

			ln, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			_, port, err := net.SplitHostPort(ln.Addr().String())
			if err != nil {
				return fmt.Errorf("failed to parse address: %v", err)
			}
			go server.Serve(ln)

			sshCommand := exec.Command("ssh", "-R", fmt.Sprintf("80:localhost:%s", port), "nokey@localhost.run")
			sshCommand.Stdin = os.Stdin
			sshCommand.Stdout = os.Stdout
			sshCommand.Stderr = os.Stderr
			if err := sshCommand.Run(); err != nil {
				return fmt.Errorf("failed to start ssh command: %v", err)
			}

			return nil
		},
	}

	return cmd
}
