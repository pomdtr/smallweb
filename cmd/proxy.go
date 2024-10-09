package cmd

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

func NewCmdProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy <http-addr> <ssh-addr>",
		Args:  cobra.ExactArgs(2),
		Short: "Start a proxy server",
		RunE: func(cmd *cobra.Command, args []string) error {
			httpAddr, sshAddr := args[0], args[1]
			forwardHandler := &ForwardedTCPHandler{}

			httpServer := &http.Server{
				Addr: httpAddr,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					parts := strings.SplitN(r.Host, ".", 2)
					if len(parts) != 2 {
						http.Error(w, "invalid host", http.StatusBadRequest)
						return
					}

					username := parts[0]
					ln, ok := forwardHandler.Forwards[username]
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

				}),
			}

			sshServer := ssh.Server{
				Addr: sshAddr,
				LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
					log.Println("Accepted forward", dhost, dport)
					return true
				}),
				Handler: ssh.Handler(func(s ssh.Session) {
					io.WriteString(s, "Remote forwarding available...\n")
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

			fmt.Fprintln(os.Stderr, "Starting http server on", httpAddr)
			go httpServer.ListenAndServe()
			fmt.Fprintln(os.Stderr, "Starting ssh server on", sshAddr)
			go sshServer.ListenAndServe()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, os.Interrupt)
			<-sigs
			httpServer.Close()
			sshServer.Close()
			return nil
		},
	}

	return cmd
}

const (
	forwardedTCPChannelType = "forwarded-tcpip"
)

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

func (h *ForwardedTCPHandler) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	h.Lock()
	if h.Forwards == nil {
		h.Forwards = make(map[string]net.Listener)
	}
	h.Unlock()

	username := ctx.User()
	if username == "" {
		return false, []byte("no username")
	}

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
		h.Forwards[username] = ln
		h.Unlock()
		go func() {
			<-ctx.Done()
			h.Lock()
			ln, ok := h.Forwards[username]
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
					ch, reqs, err := conn.OpenChannel(forwardedTCPChannelType, payload)
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
			delete(h.Forwards, username)
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
		ln, ok := h.Forwards[username]
		h.Unlock()
		if ok {
			ln.Close()
		}
		return true, nil
	default:
		return false, nil
	}
}
