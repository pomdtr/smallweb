package worker

import (
	"bufio"
	"context"
	"crypto"
	_ "embed"
	"encoding/base64"
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
	"runtime"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/gorilla/websocket"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/config"
	"github.com/pomdtr/smallweb/meta"
	"github.com/pomdtr/smallweb/utils"
)

//go:embed sandbox.ts
var sandboxBytes []byte
var sandboxPath = filepath.Join(xdg.CacheHome, "smallweb", "sandbox", fmt.Sprintf("%s.ts", hash(sandboxBytes)))

func hash(b []byte) string {
	sha := crypto.SHA256.New()
	sha.Write(b)
	return base64.URLEncoding.EncodeToString(sha.Sum(nil))
}

func init() {
	if !utils.FileExists(sandboxPath) {
		if err := os.MkdirAll(filepath.Dir(sandboxPath), 0755); err != nil {
			log.Fatalf("could not create directory: %v", err)
		}

		if err := os.WriteFile(sandboxPath, sandboxBytes, 0644); err != nil {
			log.Fatalf("could not write file: %v", err)
		}
	}
}

type Worker struct {
	App app.App
	*slog.Logger
	Env map[string]string
}

func NewWorker(app app.App, conf config.Config) *Worker {
	worker := &Worker{
		App: app,
	}

	worker.Env = make(map[string]string)

	worker.Env["DENO_NO_UPDATE_CHECK"] = "1"
	worker.Env["DENO_DIR"] = filepath.Join(xdg.CacheHome, "smallweb", "deno", "dir")
	worker.Env["TMPDIR"] = filepath.Join(app.Root(), "data", "tmp")

	worker.Env["SMALLWEB_VERSION"] = meta.Version
	worker.Env["SMALLWEB_DOMAIN"] = conf.Domain
	worker.Env["SMALLWEB_DIR"] = utils.RootDir()
	worker.Env["SMALLWEB_APP_NAME"] = app.Name
	worker.Env["SMALLWEB_APP_URL"] = app.URL

	return worker
}

var upgrader = websocket.Upgrader{} // use default options

func (me *Worker) Flags(execPath string) []string {
	flags := []string{
		"--allow-net",
		"--allow-import",
		"--allow-env",
		"--allow-sys=osRelease,homedir,cpus,hostname",
		"--unstable-kv",
		"--no-prompt",
		"--quiet",
	}

	if me.App.Config.Admin {
		flags = append(
			flags,
			fmt.Sprintf("--allow-read=%s,%s,%s,%s", utils.RootDir(), me.Env["DENO_DIR"], sandboxPath, execPath),
			fmt.Sprintf("--allow-write=%s", utils.RootDir()),
		)
	} else {
		flags = append(
			flags,
			fmt.Sprintf("--allow-read=%s,%s,%s,%s", me.App.Root(), me.Env["DENO_DIR"], sandboxPath, execPath),
			fmt.Sprintf("--allow-write=%s", filepath.Join(me.App.Root(), "data")),
		)
	}

	if configPath := filepath.Join(me.App.Dir, "deno.json"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	} else if configPath := filepath.Join(me.App.Dir, "deno.jsonc"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	}

	return flags
}

func (me *Worker) Start() (*exec.Cmd, int, error) {
	port, err := GetFreePort()
	if err != nil {
		return nil, 0, fmt.Errorf("could not get free port: %w", err)
	}

	deno, err := DenoExecutable()
	if err != nil {
		return nil, 0, fmt.Errorf("could not find deno executable")
	}

	args := []string{"run"}
	args = append(args, me.Flags(deno)...)
	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
		"command":    "fetch",
		"entrypoint": me.App.Entrypoint(),
		"port":       port,
	})
	args = append(args, sandboxPath, input.String())

	command := exec.Command(deno, args...)
	command.Dir = me.App.Root()

	for k, v := range me.Env {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range me.App.Env {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		return nil, 0, fmt.Errorf("could not get stdout pipe: %w", err)
	}

	stderrPipe, err := command.StderrPipe()
	if err != nil {
		return nil, 0, fmt.Errorf("could not get stderr pipe: %w", err)
	}

	if err := command.Start(); err != nil {
		return nil, 0, fmt.Errorf("could not start server: %w", err)
	}

	// Wait for the "READY" signal from stdout
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Scan()
	if scanner.Text() != "READY" {
		return nil, 0, fmt.Errorf("server did not start correctly")
	}

	// Function to handle logging for both stdout and stderr
	logPipe := func(pipe io.ReadCloser, logType string) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			if me.Logger == nil {
				os.Stdout.WriteString(scanner.Text() + "\n")
				continue
			}

			me.Logger.LogAttrs(
				context.Background(),
				slog.LevelInfo,
				logType,
				slog.String("type", logType),
				slog.String("app", me.App.Name),
				slog.String("text", scanner.Text()),
			)
		}
	}

	// Start goroutine for stdout
	go logPipe(stdoutPipe, "stdout")

	// Start goroutine for stderr
	go logPipe(stderrPipe, "stderr")

	return command, port, nil
}

func (me *Worker) Stop(command *exec.Cmd) error {
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

func (me *Worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	command, port, err := me.Start()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer me.Stop(command)

	// handle websockets
	if r.Header.Get("Upgrade") == "websocket" {
		serverConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading connection: %v", err)
			return
		}
		defer serverConn.Close()

		clientConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d%s", port, r.URL.Path), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer clientConn.Close()

		// Channel to signal closure
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				messageType, p, err := clientConn.ReadMessage()
				if err != nil {
					return
				}

				if err := serverConn.WriteMessage(messageType, p); err != nil {
					return
				}
			}
		}()

		go func() {
			defer close(done)
			for {
				messageType, p, err := serverConn.ReadMessage()
				if err != nil {
					return
				}

				if err := clientConn.WriteMessage(messageType, p); err != nil {
					return
				}
			}
		}()

		<-done
		return
	}

	request, err := http.NewRequest(r.Method, fmt.Sprintf("http://127.0.0.1:%d%s", port, r.URL.String()), r.Body)
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

	flusher := w.(http.Flusher)
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
	} {
		if utils.FileExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("deno executable not found")
}

func (me *Worker) Command(args ...string) (*exec.Cmd, error) {
	if args == nil {
		args = []string{}
	}

	deno, err := DenoExecutable()
	if err != nil {
		return nil, fmt.Errorf("could not find deno executable")
	}

	denoArgs := []string{"run"}
	denoArgs = append(denoArgs, me.Flags(deno)...)
	if runtime.GOOS == "darwin" {
		denoArgs = append(denoArgs, "--allow-run=open")
	} else {
		denoArgs = append(denoArgs, "--allow-run=xdg-open")
	}

	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
		"command":    "run",
		"entrypoint": me.App.Entrypoint(),
		"args":       args,
	})
	denoArgs = append(denoArgs, sandboxPath, input.String())

	command := exec.Command(deno, denoArgs...)
	command.Dir = me.App.Root()

	command.Env = os.Environ()
	for k, v := range me.Env {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range me.App.Env {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", k, v))
	}

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
