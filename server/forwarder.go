package server

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/pomdtr/smallweb/server/storage"
	gossh "golang.org/x/crypto/ssh"
)

// Forwarder can be enabled by creating a Forwarder and
// adding the HandleSSHRequest callback to the server's RequestHandlers under
// tcpip-forward and cancel-tcpip-forward.
type Forwarder struct {
	db    *storage.DB
	ports map[string]int
	conns map[string]gossh.Conn
	sync.Mutex
}

func NewForwarder(db *storage.DB) *Forwarder {
	return &Forwarder{
		db:    db,
		ports: make(map[string]int),
		conns: make(map[string]gossh.Conn),
	}
}

func (me *Forwarder) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	user, err := me.db.UserFromContext(ctx)
	if err != nil {
		slog.Info("no user found", slog.String("error", err.Error()))
		return false, nil
	}

	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)

	freeport, err := GetFreePort()
	if err != nil {
		return false, nil
	}

	addr := fmt.Sprintf("127.0.0.1:%d", freeport)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false, nil
	}

	me.Lock()
	me.ports[user.Name] = freeport
	me.conns[user.Name] = conn
	me.Unlock()
	go func() {
		<-ctx.Done()
		me.Lock()
		_, ok := me.ports[user.Name]
		me.Unlock()
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

			go func() {
				ch, reqs, err := conn.OpenChannel("forwarded-smallweb", nil)
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
		me.Lock()
		delete(me.ports, addr)
		delete(me.conns, addr)
		me.Unlock()
	}()
	return true, nil
}

func (me *Forwarder) SendRequest(user string, reqType string, payload []byte) (bool, []byte, error) {
	conn, ok := me.conns[user]
	if !ok {
		return false, nil, fmt.Errorf("no connection found")
	}

	ok, payload, err := conn.SendRequest(reqType, true, payload)
	if err != nil {
		return false, nil, err
	}

	return ok, payload, nil
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
