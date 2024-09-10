package term

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

//go:embed dist
var embedFs embed.FS

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func StripAnsi(b []byte) []byte {
	return re.ReplaceAll(b, nil)
}

type Handler struct {
	Name       string
	Args       []string
	Dir        string
	Env        []string
	fileServer http.Handler
	lock       sync.Mutex
	ttys       map[string]*os.File
}

type ResizePayload struct {
	ID   string `json:"id"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func NewHandler(name string, args ...string) (*Handler, error) {
	subFS, err := fs.Sub(embedFs, "frontend/dist")
	if err != nil {
		return nil, fmt.Errorf("failed to get sub fs: %w", err)
	}

	return &Handler{
		Name:       name,
		Args:       args,
		fileServer: http.FileServer(http.FS(subFS)),
		ttys:       make(map[string]*os.File),
	}, nil
}

func (me *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.Header.Get("Upgrade") == "websocket" {
			me.HandleWebSocket(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/assets/") {
			me.fileServer.ServeHTTP(w, r)
			return
		}

		http.StripPrefix(r.URL.Path, me.fileServer).ServeHTTP(w, r)
	case http.MethodPatch:
		defer r.Body.Close()

		var payload ResizePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		me.lock.Lock()
		defer me.lock.Unlock()

		tty, ok := me.ttys[payload.ID]
		if !ok {
			http.Error(w, "tty not found", http.StatusNotFound)
			return
		}

		if err := pty.Setsize(tty, &pty.Winsize{Cols: uint16(payload.Cols), Rows: uint16(payload.Rows)}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		defer r.Body.Close()

		cmd := exec.Command(me.Name, me.Args...)
		cmd.Args = append(cmd.Args, extractArgs(r.URL)...)
		cmd.Stdin = r.Body
		cmd.Dir = me.Dir
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, "NO_COLOR=1")
		cmd.Env = append(cmd.Env, "CI=1")

		output, err := cmd.Output()
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(StripAnsi(exitError.Stderr))
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(StripAnsi(output))
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}

}

func extractArgs(url *url.URL) []string {
	var args []string
	if url.Path != "/" {
		args = strings.Split(url.Path[1:], "/")
	}
	for key, values := range url.Query() {
		if key == "_payload" {
			continue
		}

		value := values[0]

		if len(key) == 1 {
			if value == "" {
				args = append(args, fmt.Sprintf("-%s", key))
			} else {
				args = append(args, fmt.Sprintf("-%s=%s", key, value))
			}
		} else {
			if value == "" {
				args = append(args, fmt.Sprintf("--%s", key))
			} else {
				args = append(args, fmt.Sprintf("--%s=%s", key, value))
			}
		}
	}

	return args
}

var (
	maxBufferSizeBytes   = 512
	keepalivePingTimeout = 20 * time.Second
)

func (me *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	payloadString := r.URL.Query().Get("_payload")
	if payloadString == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing payload"))
		return
	}

	var payload ResizePayload
	if err := json.Unmarshal([]byte(payloadString), &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("failed to parse payload: %s", err)))
		return
	}

	cmd := exec.Command(me.Name, me.Args...)
	cmd.Args = append(cmd.Args, extractArgs(r.URL)...)
	cmd.Dir = me.Dir
	cmd.Env = me.Env
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(payload.Cols), Rows: uint16(payload.Rows)})
	if err != nil {
		log.Printf("failed to start pty: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed to start pty: %s", err)))
		return
	}

	defer func() {
		if err := tty.Close(); err != nil {
			log.Printf("failed to close tty: %s", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-time.After(5 * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("failed to kill command: %s", err)
			}
			log.Println("command timed out")
		case err := <-done:
			if err != nil {
				log.Printf("failed to wait for command: %s", err)
			}
		}
	}()

	me.lock.Lock()
	me.ttys[payload.ID] = tty
	me.lock.Unlock()

	upgrader := getConnectionUpgrader(maxBufferSizeBytes)
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection: %s", err)
		return
	}
	defer connection.Close()

	var waiter sync.WaitGroup
	waiter.Add(1)

	// tty << xterm.js
	go func() {
		for {
			// data processing
			_, data, err := connection.ReadMessage()
			if err != nil {
				// log.Printf("failed to get next reader: %s", err)
				waiter.Done()
				return
			}
			dataLength := len(data)
			dataBuffer := bytes.Trim(data, "\x00")

			// process
			if dataLength == -1 { // invalid
				log.Printf("failed to get the correct number of bytes read, ignoring message")
				continue
			}

			// write to tty
			if _, err := tty.Write(dataBuffer); err != nil {
				// log.Printf("failed to write %v bytes to tty: %s", len(dataBuffer), err)
				continue
			}
		}
	}()

	messages := make(chan []byte)
	// tty >> xterm.js
	go func() {
		for {
			buffer := make([]byte, maxBufferSizeBytes)
			readLength, err := tty.Read(buffer)
			if err != nil {
				connection.Close()
				// log.Printf("failed to read from tty: %s", err)
				return
			}

			messages <- buffer[:readLength]
		}
	}()

	lastPingTime := time.Now()
	connection.SetPongHandler(func(appData string) error {
		lastPingTime = time.Now()
		return nil
	})

	// this is a keep-alive loop that ensures connection does not hang-up itself
	go func() {
		ticker := time.NewTicker(keepalivePingTimeout / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				connection.WriteMessage(websocket.PingMessage, []byte("keepalive"))
				if time.Since(lastPingTime) > keepalivePingTimeout {
					// log.Printf("connection timeout, closing connection")
					connection.Close()
					return
				}
			case m := <-messages:
				if err := connection.WriteMessage(websocket.BinaryMessage, m); err != nil {
					// log.Printf("failed to send %v bytes from tty to xterm.js", len(m))
					continue
				}
			}
		}
	}()

	waiter.Wait()

	me.lock.Lock()
	delete(me.ttys, r.RemoteAddr)
	me.lock.Unlock()
}

func getConnectionUpgrader(
	maxBufferSizeBytes int,
) websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		HandshakeTimeout: 0,
		ReadBufferSize:   maxBufferSizeBytes,
		WriteBufferSize:  maxBufferSizeBytes,
	}
}
