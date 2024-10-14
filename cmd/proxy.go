package cmd

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

func NewCmdProxy() *cobra.Command {
	var flags struct {
		httpPort int
		sshPort  int
	}

	cmd := &cobra.Command{
		Use:    "proxy",
		Args:   cobra.NoArgs,
		Hidden: true,
		Short:  "Start a proxy server",
		RunE: func(cmd *cobra.Command, args []string) error {
			forwardHandler := &ForwardedTCPHandler{}

			httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ln, ok := forwardHandler.GetListener(r.Host)
				if !ok {
					http.Error(w, "user not connected", http.StatusNotFound)
					return
				}

				client := &http.Client{
					Transport: &http.Transport{
						Dial: func(network, addr string) (net.Conn, error) {
							return net.Dial("tcp", ln.Addr().String())
						},
					},
				}

				url := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())
				req, err := http.NewRequest(r.Method, url, r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				for k, v := range r.Header {
					for _, vv := range v {
						req.Header.Add(k, vv)
					}
				}

				resp, err := client.Do(req)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.WriteHeader(resp.StatusCode)
				for k, v := range resp.Header {
					for _, vv := range v {
						w.Header().Add(k, vv)
					}
				}
				io.Copy(w, resp.Body)

			})

			sshServer := ssh.Server{
				Addr: fmt.Sprintf(":%d", flags.sshPort),
				LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
					log.Println("Accepted forward", dhost, dport)
					return true
				}),
				Handler: ssh.Handler(func(s ssh.Session) {
					fmt.Fprintln(s, s.Context().SessionID())
					select {}
				}),
				ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
					log.Println("attempt to bind", host, port, "granted")
					return true
				}),
				RequestHandlers: map[string]ssh.RequestHandler{
					"tcpip-forward":        forwardHandler.HandleSSHRequest,
					"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
				},
			}

			go sshServer.ListenAndServe()

			fmt.Fprintln(cmd.OutOrStdout(), "Forwarding requests from HTTP port", flags.httpPort, "to SSH port", flags.sshPort, "...")
			return http.ListenAndServe(fmt.Sprintf(":%d", flags.httpPort), http.HandlerFunc(httpHandler))
		},
	}

	cmd.Flags().IntVar(&flags.httpPort, "http-port", 8080, "HTTP port to listen on")
	cmd.Flags().IntVar(&flags.sshPort, "ssh-port", 2222, "SSH port to listen on")

	return cmd
}

type remoteForwardRequest struct {
	BindAddr string
	BindPort uint32
}

type remoteForwardSuccess struct {
	BindPort uint32
}

type remoteForwardCancelRequest struct {
	BindAddr string
	BindPort uint32
}

type remoteForwardChannelData struct {
	DestAddr   string
	DestPort   uint32
	OriginAddr string
	OriginPort uint32
}

// ForwardedTCPHandler can be enabled by creating a ForwardedTCPHandler and
// adding the HandleSSHRequest callback to the server's RequestHandlers under
// tcpip-forward and cancel-tcpip-forward.
type ForwardedTCPHandler struct {
	Forwards map[string]net.Listener
	sync.Mutex
}

func (h *ForwardedTCPHandler) GetListener(host string) (net.Listener, bool) {
	h.Lock()
	defer h.Unlock()
	if ln, ok := h.Forwards[host]; ok {
		return ln, true
	}

	parts := strings.SplitN(host, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}

	ln, ok := h.Forwards[parts[1]]
	return ln, ok
}

func SessionKey(ctx ssh.Context) string {
	return ctx.SessionID()[0:8]
}

func (h *ForwardedTCPHandler) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	h.Lock()
	if h.Forwards == nil {
		h.Forwards = make(map[string]net.Listener)
	}
	h.Unlock()

	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
	switch req.Type {
	case "tcpip-forward":
		var reqPayload remoteForwardRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			// TODO: log parse failure
			return false, []byte{}
		}

		if srv.ReversePortForwardingCallback == nil || !srv.ReversePortForwardingCallback(ctx, reqPayload.BindAddr, reqPayload.BindPort) {
			return false, []byte("port forwarding is disabled")
		}

		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return false, []byte{}
		}

		ln, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return false, []byte{}
		}

		_, destPortStr, _ := net.SplitHostPort(ln.Addr().String())
		destPort, _ := strconv.Atoi(destPortStr)
		h.Lock()
		if _, ok := h.Forwards[SessionKey(ctx)]; ok {
			h.Unlock()
			ln.Close()
			return false, []byte("forward already exists")
		}

		h.Forwards[SessionKey(ctx)] = ln
		h.Unlock()
		go func() {
			<-ctx.Done()
			h.Lock()
			ln, ok := h.Forwards[SessionKey(ctx)]
			h.Unlock()
			if ok {
				ln.Close()
			}
		}()
		go func() {
			for {
				c, err := ln.Accept()
				if err != nil {
					// TODO: log accept failure
					break
				}
				originAddr, originPortStr, _ := net.SplitHostPort(c.RemoteAddr().String())
				originPort, _ := strconv.Atoi(originPortStr)
				payload := gossh.Marshal(&remoteForwardChannelData{
					DestAddr:   reqPayload.BindAddr,
					DestPort:   reqPayload.BindPort,
					OriginAddr: originAddr,
					OriginPort: uint32(originPort),
				})
				go func() {
					ch, reqs, err := conn.OpenChannel("forwarded-tcpip", payload)
					if err != nil {
						// TODO: log failure to open channel
						log.Println(err)
						c.Close()
						return
					}
					go gossh.DiscardRequests(reqs)
					go func() {
						defer ch.Close()
						defer c.Close()
						io.Copy(ch, c)
					}()
					go func() {
						defer ch.Close()
						defer c.Close()
						io.Copy(c, ch)
					}()
				}()
			}
			h.Lock()
			delete(h.Forwards, SessionKey(ctx))
			h.Unlock()
		}()
		return true, gossh.Marshal(&remoteForwardSuccess{uint32(destPort)})

	case "cancel-tcpip-forward":
		var reqPayload remoteForwardCancelRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			// TODO: log parse failure
			return false, []byte{}
		}
		h.Lock()
		ln, ok := h.Forwards[SessionKey(ctx)]
		h.Unlock()
		if ok {
			ln.Close()
		}
		return true, nil
	default:
		return false, nil
	}
}
