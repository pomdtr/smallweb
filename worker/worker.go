package worker

import (
	"bufio"
	"crypto"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/gorilla/websocket"
	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
)

//go:embed sandbox.ts
var sandboxBytes []byte
var sandboxPath string

func init() {
	sha := crypto.SHA256.New()
	sha.Write(sandboxBytes)
	sandboxHash := base64.URLEncoding.EncodeToString(sha.Sum(nil))
	sandboxPath = filepath.Join(xdg.DataHome, "smallweb", string(sandboxHash), "sandbox.ts")
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
	App  app.App
	Env  map[string]string
	port int
	cmd  *exec.Cmd
}

func NewWorker(app app.App, env map[string]string) *Worker {
	if env == nil {
		env = make(map[string]string)
	}

	worker := &Worker{
		App: app,
		Env: env,
	}

	worker.Env["DENO_NO_UPDATE_CHECK"] = "1"
	worker.Env["DENO_DIR"] = filepath.Join(os.Getenv("HOME"), ".cache", "smallweb", "deno")
	for k, v := range env {
		worker.Env[k] = v
	}

	return worker
}

var upgrader = websocket.Upgrader{} // use default options

func (me *Worker) Flags() []string {
	flags := []string{
		"--allow-net",
		"--allow-env",
		"--allow-sys=osRelease,homedir,cpus,hostname",
		"--unstable-kv",
		"--no-prompt",
		"--quiet",
		fmt.Sprintf("--location=%s", me.App.Url),
		fmt.Sprintf("--allow-read=%s,%s,%s", me.App.Root(), me.Env["DENO_DIR"], sandboxPath),
		fmt.Sprintf("--allow-write=%s", me.App.Root()),
	}

	if configPath := filepath.Join(me.App.Dir, "deno.json"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	} else if configPath := filepath.Join(me.App.Dir, "deno.jsonc"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	}

	return flags
}

func (me *Worker) StartServer() error {
	port, err := GetFreePort()
	if err != nil {
		return fmt.Errorf("could not get free port: %w", err)
	}
	me.port = port

	args := []string{"run"}
	args = append(args, me.Flags()...)

	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
		"command":    "fetch",
		"entrypoint": me.App.Entrypoint(),
		"port":       port,
	})
	args = append(args, sandboxPath, input.String())

	deno, err := DenoExecutable()
	if err != nil {
		return fmt.Errorf("could not find deno executable")
	}

	me.cmd = exec.Command(deno, args...)
	me.cmd.Dir = me.App.Root()

	var env []string
	for k, v := range me.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range me.App.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	me.cmd.Env = env

	stdout, err := me.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not get stdout pipe: %w", err)
	}

	me.cmd.Stderr = os.Stderr
	if err := me.cmd.Start(); err != nil {
		return fmt.Errorf("could not start server: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Scan()
	line := scanner.Text()
	if !(line == "READY") {
		return fmt.Errorf("server did not start correctly")
	}

	go func() {
		for scanner.Scan() {
			os.Stdout.WriteString(scanner.Text() + "\n")
		}
	}()

	return nil
}

func (me *Worker) StopServer() error {
	if err := me.cmd.Process.Signal(os.Interrupt); err != nil {
		log.Printf("Failed to send interrupt signal: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- me.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		if err := me.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}

		me.cmd = nil
		me.port = 0
		return fmt.Errorf("process did not exit after 5 seconds")
	case <-done:
		me.cmd = nil
		me.port = 0
		return nil
	}
}

func (me *Worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://%s%s", r.Host, r.URL.String())

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

		go func() {
			for {
				messageType, p, err := clientConn.ReadMessage()
				if err != nil {
					log.Printf("Error reading message: %v", err)
					break
				}

				if err := serverConn.WriteMessage(messageType, p); err != nil {
					log.Printf("Error writing message: %v", err)
					break
				}
			}
		}()

		for {
			messageType, p, err := serverConn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				break
			}

			if err := clientConn.WriteMessage(messageType, p); err != nil {
				log.Printf("Error writing message: %v", err)
				break
			}
		}
	}

	request, err := http.NewRequest(r.Method, fmt.Sprintf("http://localhost:%d%s", me.port, r.URL.String()), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range r.Header {
		for _, vv := range v {
			request.Header.Add(k, vv)
		}
	}

	request.Header.Add("X-Smallweb-Url", url)
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

	denoArgs := []string{"run"}
	denoArgs = append(denoArgs, me.Flags()...)

	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
		"command":    "run",
		"entrypoint": me.App.Entrypoint(),
		"args":       args,
	})
	denoArgs = append(denoArgs, sandboxPath, input.String())
	deno, err := DenoExecutable()
	if err != nil {
		return nil, fmt.Errorf("could not find deno executable")
	}

	cmd := exec.Command(deno, denoArgs...)
	cmd.Dir = me.App.Root()
	for k, v := range me.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range me.App.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	return cmd, nil
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
