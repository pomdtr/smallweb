package worker

import (
	"bufio"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adrg/xdg"
	"github.com/gorilla/websocket"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/build"
	"github.com/pomdtr/smallweb/internal/utils"
)

//go:embed sandbox.ts
var sandboxBytes []byte
var sandboxPath = filepath.Join(xdg.CacheHome, "smallweb", "sandbox", build.Version, "mod.ts")

func init() {
	if err := os.MkdirAll(filepath.Dir(sandboxPath), 0o755); err != nil {
		panic(fmt.Errorf("could not create sandbox directory: %w", err))
	}

	if err := os.WriteFile(sandboxPath, sandboxBytes, 0o644); err != nil {
		panic(fmt.Errorf("could not write sandbox file: %w", err))
	}
}

type Worker struct {
	App        app.App
	StartedAt  time.Time
	apiHandler http.Handler

	port           int
	idleTimer      *time.Timer
	command        *exec.Cmd
	activeRequests atomic.Int32
}

func NewWorker(app app.App, apiHandler http.Handler) *Worker {
	worker := &Worker{
		App:        app,
		apiHandler: apiHandler,
	}

	return worker
}

var upgrader = websocket.Upgrader{} // use default options

type SandboxMethod string

func (me *Worker) DenoArgs(socketPath string) ([]string, error) {
	args := []string{
		"--allow-net",
		"--allow-import",
		"--allow-env",
		"--allow-sys",
		"--no-prompt",
		"--quiet",
	}

	for _, configName := range []string{"deno.json", "deno.jsonc"} {
		configPath := filepath.Join(me.App.Root(), configName)
		if _, err := os.Stat(configPath); err == nil {
			args = append(args, fmt.Sprintf("--config=%s", configPath))
			break
		}
	}

	npmCache := filepath.Join(xdg.CacheHome, "deno", "npm", "registry.npmjs.org")

	args = append(
		args,
		fmt.Sprintf("--allow-read=%s,%s,%s", me.App.Root(), npmCache, socketPath),
		fmt.Sprintf("--allow-write=%s,%s", me.App.DataDir(), socketPath),
	)

	return args, nil
}

func (me *Worker) Start(logger *slog.Logger) error {
	port, err := GetFreePort()
	if err != nil {
		return fmt.Errorf("could not get free port: %w", err)
	}
	me.port = port

	deno, err := DenoExecutable()
	if err != nil {
		return fmt.Errorf("could not find deno executable")
	}

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("smallweb-%s.sock", rand.Text()))
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not create unix socket: %w", err)
	}

	go http.Serve(ln, me.apiHandler)

	args := []string{"run"}
	denoArgs, err := me.DenoArgs(socketPath)
	if err != nil {
		return fmt.Errorf("could not get deno args: %w", err)
	}

	args = append(args, denoArgs...)
	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]any{
		"command":    "fetch",
		"entrypoint": me.App.Entrypoint(),
		"port":       port,
	}); err != nil {
		return fmt.Errorf("could not encode input: %w", err)
	}

	args = append(args, sandboxPath, input.String())

	command := exec.Command(deno, args...)
	command.Dir = me.App.Root()
	command.Env = me.Env(socketPath)

	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not get stdout pipe: %w", err)
	}

	stderrPipe, err := command.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not get stderr pipe: %w", err)
	}

	if err := command.Start(); err != nil {
		return fmt.Errorf("could not start server: %w", err)
	}

	readyChan := make(chan bool)
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			if scanner.Text() == "READY" {
				readyChan <- true
				return
			}

			os.Stderr.WriteString(scanner.Text() + "\n")
		}

		readyChan <- false
	}()

	select {
	case ready := <-readyChan:
		if !ready {
			return fmt.Errorf("server did not start correctly")
		}
	case <-time.After(30 * time.Second):
		return fmt.Errorf("server start timed out")
	}

	// Function to handle logging for both stdout and stderr
	logPipe := func(pipe io.ReadCloser, logger *slog.Logger) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			logger.Info(
				scanner.Text(),
			)
		}
	}

	if logger != nil {
		// Start goroutine for stdout
		go logPipe(stdoutPipe, logger.With("stream", "stdout"))

		// Start goroutine for stderr
		go logPipe(stderrPipe, logger.With("stream", "stderr"))
	}

	me.command = command
	me.StartedAt = time.Now()
	me.idleTimer = time.NewTimer(10 * time.Second)
	go func() {
		for {
			<-me.idleTimer.C
			// if there are no active requests, stop the worker
			if me.activeRequests.Load() == 0 {
				defer ln.Close()
				defer os.Remove(socketPath)
				_ = me.Stop()
				return
			} else {
				me.idleTimer.Reset(10 * time.Second)
			}
		}
	}()

	return nil
}

func (me *Worker) IsRunning() bool {
	return me.command != nil
}

func (me *Worker) Stop() error {
	if !me.IsRunning() {
		return nil
	}

	command := me.command
	me.command = nil

	if err := command.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		if err := command.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		return fmt.Errorf("process did not exit after 5 seconds")
	case <-done:
		return nil
	}
}

func (me *Worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	me.activeRequests.Add(1)
	defer func() {
		me.idleTimer.Reset(10 * time.Second)
		me.activeRequests.Add(-1)
	}()

	// handle websockets
	if r.Header.Get("Upgrade") == "websocket" {
		serverConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer serverConn.Close()

		clientConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d%s", me.port, r.URL.Path), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer clientConn.Close()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Channel to signal closure
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					messageType, p, err := clientConn.ReadMessage()
					if err != nil {
						return
					}

					if err := serverConn.WriteMessage(messageType, p); err != nil {
						return
					}
				}
			}
		}()

		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					messageType, p, err := serverConn.ReadMessage()
					if err != nil {
						return
					}

					if err := clientConn.WriteMessage(messageType, p); err != nil {
						return
					}
				}
			}
		}()

		wg.Wait()
		return
	}

	request, err := http.NewRequestWithContext(r.Context(), r.Method, fmt.Sprintf("http://127.0.0.1:%d%s", me.port, r.URL.String()), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range r.Header {
		for _, vv := range v {
			request.Header.Add(k, vv)
		}
	}

	if request.Header.Get("x-forwarded-host") == "" {
		request.Header.Set("x-forwarded-host", r.Host)
	}

	if request.Header.Get("x-forwarded-proto") == "" {
		if r.TLS != nil {
			request.Header.Set("x-forwarded-proto", "https")
		} else {
			request.Header.Set("x-forwarded-proto", "http")
		}
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Minute,
				KeepAlive: 5 * time.Minute,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Minute,
			ResponseHeaderTimeout: 5 * time.Minute,
		},
	}

	resp, err := client.Do(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "expected http.ResponseWriter to be an http.Flusher", http.StatusInternalServerError)
		return
	}

	// Stream the response body to the client
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				return
			}
			flusher.Flush() // flush the buffer to the client
		}
		if err != nil {
			if err != io.EOF {
				http.Error(w, "Error reading response body", http.StatusInternalServerError)
			}
			break
		}
	}
}

func DenoExecutable() (string, error) {
	if env, ok := os.LookupEnv("DENO_EXEC_PATH"); ok {
		return env, nil
	}

	if denoPath, err := exec.LookPath("deno"); err == nil {
		return denoPath, nil
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	for _, candidate := range []string{
		filepath.Join(homedir, ".deno", "bin", "deno"),
		"/home/linuxbrew/.linuxbrew/bin/deno",
		"/opt/homebrew/bin/deno",
		"/usr/local/bin/deno",
		"/usr/bin/deno",
	} {
		if utils.FileExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("deno executable not found")
}

type RunParams struct {
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
}

func (me *Worker) Run(ctx context.Context, params RunParams) error {
	args := params.Args
	if args == nil {
		args = []string{}
	}

	deno, err := DenoExecutable()
	if err != nil {
		return fmt.Errorf("could not find deno executable")
	}

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("smallweb-%s.sock", rand.Text()))
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not create unix socket: %w", err)
	}

	go http.Serve(ln, me.apiHandler)
	defer ln.Close()
	defer os.Remove(socketPath)

	cmdArgs := []string{"run"}
	denoArgs, err := me.DenoArgs(socketPath)
	if err != nil {
		return fmt.Errorf("could not get deno args: %w", err)
	}
	cmdArgs = append(cmdArgs, denoArgs...)

	payload := strings.Builder{}
	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]any{
		"command":    "run",
		"entrypoint": me.App.Entrypoint(),
		"args":       args,
	}); err != nil {
		return fmt.Errorf("could not encode input: %w", err)
	}

	cmdArgs = append(cmdArgs, sandboxPath, payload.String())

	command := exec.CommandContext(ctx, deno, cmdArgs...)
	command.Dir = me.App.Root()
	command.Stdout = params.Stdout
	command.Stderr = params.Stderr

	command.Env = me.Env(socketPath)

	return nil
}

func (me *Worker) SendEmail(ctx context.Context, msg []byte) error {
	deno, err := DenoExecutable()
	if err != nil {
		return fmt.Errorf("could not find deno executable")
	}

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("smallweb-%s.sock", rand.Text()))
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not create unix socket: %w", err)
	}

	go http.Serve(ln, me.apiHandler)
	defer ln.Close()
	defer os.Remove(socketPath)

	args := []string{"run"}
	denoArgs, err := me.DenoArgs(socketPath)
	if err != nil {
		return fmt.Errorf("could not get deno args: %w", err)
	}

	args = append(args, denoArgs...)

	payload := strings.Builder{}
	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]any{
		"command":    "email",
		"entrypoint": me.App.Entrypoint(),
		"msg":        base64.StdEncoding.EncodeToString(msg),
	}); err != nil {
		return fmt.Errorf("could not encode input: %w", err)
	}

	denoArgs = append(args, sandboxPath, payload.String())

	command := exec.CommandContext(ctx, deno, denoArgs...)

	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	command.Dir = me.App.Root()

	command.Env = me.Env(socketPath)

	return command.Run()
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

func (me *Worker) Env(socketPath string) []string {
	env := []string{}

	for k, v := range me.App.Config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range me.App.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	env = append(env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
	env = append(env, "DENO_NO_UPDATE_CHECK=1")

	// open telemetry
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "OTEL_") {
			env = append(env, value)
		}

		if strings.HasPrefix(value, "DENO_") {
			env = append(env, value)
		}
	}

	env = append(env, fmt.Sprintf("SMALLWEB_SOCKET_PATH=%s", socketPath))
	env = append(env, fmt.Sprintf("OTEL_SERVICE_NAME=%s", me.App.Name))

	return env
}
