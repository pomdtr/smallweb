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

	"github.com/charmbracelet/ssh"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

const KeyCtrlC = 3 // ASCII ETX -> Ctrl+C
const KeyCtrlD = 4 // ASCII EOT -> Ctrl+D

func NewCmdProxy() *cobra.Command {
	var flags struct {
		httpPort int
		sshPort  int
		apiPort  int
	}

	cmd := &cobra.Command{
		Use:    "proxy",
		Args:   cobra.NoArgs,
		Hidden: true,
		Short:  "Start a proxy server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			forwardHandler := &ForwardedTCPHandler{}

			httpServer := http.Server{
				Addr: fmt.Sprintf(":%d", flags.httpPort),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ln, ok := forwardHandler.GetListener(r.Host)
					if !ok {
						http.NotFound(w, r)
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
				}),
			}

			apiServer := http.Server{
				Addr: fmt.Sprintf(":%d", flags.apiPort),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mux := http.NewServeMux()

					mux.HandleFunc("GET /check", func(w http.ResponseWriter, r *http.Request) {
						domain := r.URL.Query().Get("domain")
						if domain == "" {
							http.Error(w, "missing domain", http.StatusBadRequest)
							return
						}

						if _, ok := forwardHandler.GetListener(domain); !ok {
							http.Error(w, "no forward found", http.StatusNotFound)
							return
						}

						w.WriteHeader(http.StatusOK)
						w.Write([]byte("OK"))
					})

					mux.ServeHTTP(w, r)
				}),
			}

			sshServer := ssh.Server{
				Addr: fmt.Sprintf(":%d", flags.sshPort),
				Handler: ssh.Handler(func(s ssh.Session) {
					select {}
				}),
				ReversePortForwardingCallback: func(ctx ssh.Context, bindAddr string, bindPort uint32) bool {
					return true
				},
				RequestHandlers: map[string]ssh.RequestHandler{
					"tcpip-forward":        forwardHandler.HandleSSHRequest,
					"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
				},
			}

			go sshServer.ListenAndServe()
			go apiServer.ListenAndServe()

			fmt.Fprintln(cmd.OutOrStdout(), "Forwarding requests from HTTP Port", flags.httpPort, "to SSH port", flags.sshPort, "...")
			return httpServer.ListenAndServe()
		},
	}

	cmd.Flags().IntVar(&flags.httpPort, "http-port", 8080, "HTTP port to listen on")
	cmd.Flags().IntVar(&flags.apiPort, "api-port", 8081, "API port to listen on")
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

func (me *ForwardedTCPHandler) GetListener(host string) (net.Listener, bool) {
	if me.Forwards == nil {
		return nil, false
	}

	me.Lock()
	defer me.Unlock()

	if ln, ok := me.Forwards[host]; ok {
		return ln, true
	}

	domain := strings.SplitN(host, ".", 2)[1]
	if ln, ok := me.Forwards[domain]; ok {
		return ln, true
	}

	return nil, false
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
		if _, ok := h.Forwards[reqPayload.BindAddr]; ok {
			h.Unlock()
			ln.Close()
			return false, []byte("forward already exists")
		}

		h.Forwards[reqPayload.BindAddr] = ln
		h.Unlock()
		go func() {
			<-ctx.Done()
			h.Lock()
			ln, ok := h.Forwards[reqPayload.BindAddr]
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
			delete(h.Forwards, reqPayload.BindAddr)
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
		ln, ok := h.Forwards[reqPayload.BindAddr]
		h.Unlock()
		if ok {
			ln.Close()
		}
		return true, nil
	default:
		return false, nil
	}
}
