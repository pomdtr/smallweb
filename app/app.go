package app

import (
	"bufio"
	_ "embed"
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

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/pomdtr/smallweb/utils"
	"github.com/tailscale/hujson"
)

type AppConfig struct {
	Private    bool      `json:"private"`
	Root       string    `json:"dir"`
	Entrypoint string    `json:"entrypoint"`
	Crons      []CronJob `json:"crons"`
}

type CronJob struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Schedule    string   `json:"schedule"`
	Args        []string `json:"args"`
}

type CronJobRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

type App struct {
	Config   AppConfig
	Dir      string
	Location string
	Env      map[string]string
	port     int
	cmd      *exec.Cmd
}

func NewApp(dir string, Domain string, env map[string]string) (*App, error) {
	worker := &App{
		Dir:      dir,
		Location: fmt.Sprintf("https://%s/", Domain),
		Env:      env,
	}

	if err := worker.LoadConfig(); err != nil {
		return nil, err
	}

	if err := worker.LoadEnv(); err != nil {
		return nil, err
	}

	return worker, nil
}

func (me *App) Root() string {
	if me.Config.Root != "" {
		return filepath.Join(me.Dir, me.Config.Root)
	} else {
		return me.Dir
	}
}

var upgrader = websocket.Upgrader{} // use default options

func (me *App) Flags(sandboxPath string) []string {
	flags := []string{
		"--allow-net",
		"--allow-env",
		"--allow-sys",
		fmt.Sprintf("--allow-read=.,%s,%s", me.Env["DENO_DIR"], sandboxPath),
		"--allow-write=.",
		fmt.Sprintf("--location=%s", me.Location),
	}

	if configPath := filepath.Join(me.Dir, "deno.json"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	} else if configPath := filepath.Join(me.Dir, "deno.jsonc"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	}

	return flags
}

func (me *App) LoadConfig() error {
	if configPath := filepath.Join(me.Dir, "smallweb.json"); utils.FileExists(configPath) {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read smallweb.json: %v", err)
		}

		if err := json.Unmarshal(configBytes, &me.Config); err != nil {
			return fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return nil
	}

	if configPath := filepath.Join(me.Dir, "smallweb.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read deno.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return fmt.Errorf("could not standardize deno.jsonc: %v", err)
		}

		if err := json.Unmarshal(configBytes, &me.Config); err != nil {
			return fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return nil
	}

	if configPath := filepath.Join(me.Dir, "deno.json"); utils.FileExists(configPath) {
		denoConfigBytes, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read deno.json: %v", err)
		}

		var denoConfig map[string]json.RawMessage
		if err := json.Unmarshal(denoConfigBytes, &denoConfig); err != nil {
			return fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		configBytes, ok := denoConfig["smallweb"]
		if !ok {
			return nil
		}

		if err := json.Unmarshal(configBytes, &me.Config); err != nil {
			return fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return nil
	}

	if configPath := filepath.Join(me.Dir, "deno.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read deno.json: %v", err)
		}

		denoConfigBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return fmt.Errorf("could not standardize deno.jsonc: %v", err)
		}

		var denoConfig map[string]json.RawMessage
		if err := json.Unmarshal(denoConfigBytes, &denoConfig); err != nil {
			return fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		configBytes, ok := denoConfig["smallweb"]
		if !ok {
			return nil
		}

		if err := json.Unmarshal(configBytes, &me.Config); err != nil {
			return fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return nil
	}

	return nil
}

func (me *App) LoadEnv() error {
	me.Env["DENO_DIR"] = filepath.Join(os.Getenv("HOME"), ".cache", "smallweb", "deno")
	if dotenvPath := filepath.Join(me.Dir, ".env"); utils.FileExists(dotenvPath) {
		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			me.Env[key] = value
		}
	}

	return nil
}

func (me *App) Entrypoint() (string, error) {
	if strings.HasPrefix(me.Config.Entrypoint, "jsr:") || strings.HasPrefix(me.Config.Entrypoint, "npm:") {
		return me.Config.Entrypoint, nil
	}

	if strings.HasPrefix(me.Config.Entrypoint, "https://") || strings.HasPrefix(me.Config.Entrypoint, "http://") {
		return me.Config.Entrypoint, nil
	}

	rootDir := me.Root()
	if (me.Config.Root != "") && (me.Config.Root != ".") {
		rootDir = filepath.Join(rootDir, me.Config.Root)
	}

	if me.Config.Entrypoint != "" {
		return filepath.Join(rootDir, me.Config.Entrypoint), nil
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		path := filepath.Join(rootDir, candidate)
		if utils.FileExists(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("could not find entrypoint")
}

//go:embed sandbox.ts
var sandboxBytes []byte

func (me *App) Start() error {
	entrypoint, err := me.Entrypoint()
	if err != nil {
		return fmt.Errorf("could not get entrypoint: %w", err)
	}

	port, err := GetFreePort()
	if err != nil {
		return fmt.Errorf("could not get free port: %w", err)
	}
	me.port = port

	tempfile, err := os.CreateTemp("", "sandbox-*.ts")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %w", err)
	}

	defer os.Remove(tempfile.Name())
	if _, err := tempfile.Write(sandboxBytes); err != nil {
		return fmt.Errorf("could not write to temporary file: %w", err)
	}

	args := []string{"run"}
	args = append(args, me.Flags(tempfile.Name())...)

	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
		"command":    "serve",
		"entrypoint": entrypoint,
		"port":       port,
	})
	args = append(args, tempfile.Name(), input.String())

	deno, err := DenoExecutable()
	if err != nil {
		return fmt.Errorf("could not find deno executable")
	}

	me.cmd = exec.Command(deno, args...)
	me.cmd.Dir = me.Root()

	var env []string
	for k, v := range me.Env {
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

func (me *App) Stop() error {
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

func (me *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	start := time.Now()
	resp, err := client.Do(request)
	duration := time.Since(start)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	fmt.Fprintf(os.Stderr, "%s %s %d %dms\n", r.Method, url, resp.StatusCode, duration.Milliseconds())
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

func (me *App) Run(args []string) error {
	entrypoint, err := me.Entrypoint()
	if err != nil {
		return err
	}

	info, err := os.Stat(entrypoint)
	if err != nil {
		return fmt.Errorf("could not stat entrypoint: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("entrypoint is a directory")
	}

	tempfile, err := os.CreateTemp("", "sandbox-*.ts")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %w", err)
	}
	defer os.Remove(tempfile.Name())
	if _, err := tempfile.Write(sandboxBytes); err != nil {
		return err
	}

	denoArgs := []string{"run"}
	denoArgs = append(denoArgs, me.Flags(tempfile.Name())...)

	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
		"command":    "run",
		"entrypoint": entrypoint,
		"args":       args,
	})
	denoArgs = append(denoArgs, tempfile.Name(), input.String())
	deno, err := DenoExecutable()
	if err != nil {
		return fmt.Errorf("could not find deno executable")
	}

	cmd := exec.Command(deno, denoArgs...)
	cmd.Dir = me.Root()
	var env []string
	for k, v := range me.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
