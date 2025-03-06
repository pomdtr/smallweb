package worker

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/build"
	"github.com/pomdtr/smallweb/utils"
)

//go:embed sandbox.ts
var sandboxContent string

type Worker struct {
	App       app.App
	Env       map[string]string
	StartedAt time.Time

	port           int
	idleTimer      *time.Timer
	command        *exec.Cmd
	activeRequests atomic.Int32
}

func commandEnv(a app.App) []string {
	env := []string{}

	for k, v := range a.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	env = append(env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
	env = append(env, "DENO_NO_UPDATE_CHECK=1")
	env = append(env, fmt.Sprintf("DENO_DIR=%s", filepath.Join(xdg.CacheHome, "smallweb", "deno")))

	env = append(env, fmt.Sprintf("SMALLWEB_VERSION=%s", build.Version))
	env = append(env, fmt.Sprintf("SMALLWEB_DIR=%s", a.RootDir))
	env = append(env, fmt.Sprintf("SMALLWEB_DOMAIN=%s", a.RootDomain))
	env = append(env, fmt.Sprintf("SMALLWEB_APP_NAME=%s", a.Name))
	env = append(env, fmt.Sprintf("SMALLWEB_APP_DOMAIN=%s", a.Domain))
	env = append(env, fmt.Sprintf("SMALLWEB_APP_URL=%s", a.URL))
	env = append(env, fmt.Sprintf("SMALLWEB_APP_DIR=%s", a.BaseDir))
	if a.Admin {
		env = append(env, "SMALLWEB_ADMIN=1")
	}

	// open telemetry
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "OTEL_") {
			env = append(env, value)
		}
	}
	env = append(env, fmt.Sprintf("OTEL_SERVICE_NAME=%s", a.Domain))

	return env
}

func NewWorker(app app.App) *Worker {
	worker := &Worker{
		App: app,
	}

	return worker
}

var upgrader = websocket.Upgrader{} // use default options

type SandboxMethod string

var (
	SandboxMethodFetch SandboxMethod = "fetch"
	SandboxMethodRun   SandboxMethod = "run"
)

func (me *Worker) DenoArgs(deno string, method SandboxMethod) []string {
	args := []string{
		"--allow-net",
		"--allow-import",
		"--allow-env",
		"--allow-sys",
		"--unstable-kv",
		"--unstable-otel",
		"--unstable-temporal",
		"--node-modules-dir=none",
		"--no-prompt",
		"--quiet",
	}

	npmCache := filepath.Join(xdg.CacheHome, "smallweb", "deno", "npm", "registry.npmjs.org")
	if me.App.Admin {
		args = append(
			args,
			fmt.Sprintf("--allow-read=%s,%s,%s", me.App.RootDir, deno, npmCache),
			fmt.Sprintf("--allow-write=%s", me.App.RootDir),
			fmt.Sprintf("--deny-write=%s", filepath.Join(me.App.RootDir, ".smallweb")),
		)

		return args
	}

	// if root is not a symlink
	appDir := me.App.Dir()
	if fi, err := os.Lstat(appDir); err == nil && fi.Mode()&os.ModeSymlink == 0 {
		args = append(
			args,
			fmt.Sprintf("--allow-read=%s,%s,%s", appDir, deno, npmCache),
		)

		if method == SandboxMethodRun {
			args = append(args, fmt.Sprintf("--allow-write=%s", me.App.Dir()))
		} else {
			args = append(args, fmt.Sprintf("--allow-write=%s", me.App.DataDir()))
		}

		return args
	}

	target, err := os.Readlink(appDir)
	if err != nil {
		log.Printf("could not read symlink: %v", err)
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(appDir), target)
	}

	args = append(
		args,
		fmt.Sprintf("--allow-read=%s,%s,%s,%s", appDir, target, deno, npmCache),
	)

	if method == SandboxMethodRun {
		args = append(args, fmt.Sprintf("--allow-write=%s,%s", me.App.Dir(), target))
	} else {
		args = append(args, fmt.Sprintf("--allow-write=%s,%s", me.App.DataDir(), filepath.Join(target, "data")))
	}

	return args
}

func (me *Worker) Start() error {
	port, err := GetFreePort()
	if err != nil {
		return fmt.Errorf("could not get free port: %w", err)
	}
	me.port = port

	deno, err := DenoExecutable()
	if err != nil {
		return fmt.Errorf("could not find deno executable")
	}

	args := []string{"run"}
	args = append(args, me.DenoArgs(deno, SandboxMethodFetch)...)
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

	args = append(args, "-", input.String())

	command := exec.Command(deno, args...)
	command.Stdin = strings.NewReader(sandboxContent)
	command.Dir = me.App.Dir()
	command.Env = commandEnv(me.App)

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
	case <-time.After(5 * time.Second):
		return fmt.Errorf("server start timed out")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Function to handle logging for both stdout and stderr
	logPipe := func(pipe io.ReadCloser, stream string) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			logger.LogAttrs(
				context.Background(),
				slog.LevelInfo,
				stream,
				slog.String("type", "console"),
				slog.String("stream", stream),
				slog.String("app", me.App.Name),
				slog.String("text", scanner.Text()),
			)
		}
	}

	// Start goroutine for stdout
	go logPipe(stdoutPipe, "stdout")

	// Start goroutine for stderr
	go logPipe(stderrPipe, "stderr")

	me.command = command
	me.StartedAt = time.Now()
	me.idleTimer = time.NewTimer(10 * time.Second)
	go me.monitorIdleTimer()

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
		log.Printf("Failed to send interrupt signal: %v", err)
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

func (me *Worker) monitorIdleTimer() {
	for {
		<-me.idleTimer.C
		// if there are no active requests, stop the worker
		if me.activeRequests.Load() == 0 {
			_ = me.Stop()
			return
		}
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
			log.Printf("Error upgrading connection: %v", err)
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
		request.Header.Set("x-forwarded-proto", "https")
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Minute,
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

func (me *Worker) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if args == nil {
		args = []string{}
	}

	deno, err := DenoExecutable()
	if err != nil {
		return nil, fmt.Errorf("could not find deno executable")
	}

	denoArgs := []string{"run"}
	denoArgs = append(denoArgs, me.DenoArgs(deno, SandboxMethodRun)...)

	payload := strings.Builder{}
	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]any{
		"command":    "run",
		"entrypoint": me.App.Entrypoint(),
		"args":       args,
	}); err != nil {
		return nil, fmt.Errorf("could not encode input: %w", err)
	}

	denoArgs = append(denoArgs, "-", payload.String())

	command := exec.CommandContext(ctx, deno, denoArgs...)
	command.Stdin = strings.NewReader(sandboxContent)
	command.Dir = me.App.Dir()

	command.Env = commandEnv(me.App)

	return command, nil
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
