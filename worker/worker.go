package worker

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	Root       string    `json:"root"`
	Entrypoint string    `json:"entrypoint"`
	Crons      []CronJob `json:"crons"`
}

type CronJob struct {
	Name     string   `json:"name"`
	Schedule string   `json:"schedule"`
	Args     []string `json:"args"`
}

type CronJobRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

type App struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Url      string `json:"url"`
	Root     string `json:"root"`
}

type Worker struct {
	Config AppConfig
	App    App
	Env    map[string]string
}

func NewWorker(a App, env map[string]string) (*Worker, error) {
	config, err := LoadConfig(a.Root)
	if err != nil {
		return nil, err
	}

	worker := &Worker{
		Config: *config,
		App:    a,
		Env:    env,
	}

	envMap, err := LoadEnv(a.Root)
	if err != nil {
		return nil, err
	}

	worker.Env = envMap
	return worker, nil
}

func (me *Worker) Root() string {
	if me.Config.Root != "" {
		return filepath.Join(me.App.Root, me.Config.Root)
	} else {
		return me.App.Root
	}
}

var upgrader = websocket.Upgrader{} // use default options

func (me *Worker) Flags() []string {
	flags := []string{
		"--allow-net",
		"--allow-env",
		"--allow-sys",
		"--allow-read=.",
		"--allow-write=.",
		"--unstable-kv",
		fmt.Sprintf("--location=%s", me.App.Url),
		fmt.Sprintf("--allow-run=%s", me.Env["SMALLWEB_EXEC_PATH"]),
	}

	if configPath := filepath.Join(me.App.Root, "deno.json"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	} else if configPath := filepath.Join(me.App.Root, "deno.jsonc"); utils.FileExists(configPath) {
		flags = append(flags, "--config", configPath)
	}

	return flags
}

func LoadConfig(dir string) (*AppConfig, error) {
	var config AppConfig
	if configPath := filepath.Join(dir, "smallweb.json"); utils.FileExists(configPath) {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(dir, "smallweb.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read deno.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return nil, fmt.Errorf("could not standardize deno.jsonc: %v", err)
		}

		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(dir, "deno.json"); utils.FileExists(configPath) {
		denoConfigBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read deno.json: %v", err)
		}

		var denoConfig map[string]json.RawMessage
		if err := json.Unmarshal(denoConfigBytes, &denoConfig); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		configBytes, ok := denoConfig["smallweb"]
		if !ok {
			return &config, nil
		}

		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(dir, "deno.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read deno.json: %v", err)
		}

		denoConfigBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return nil, fmt.Errorf("could not standardize deno.jsonc: %v", err)
		}

		var denoConfig map[string]json.RawMessage
		if err := json.Unmarshal(denoConfigBytes, &denoConfig); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		configBytes, ok := denoConfig["smallweb"]
		if !ok {
			return &config, nil
		}

		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(dir, "package.json"); utils.FileExists(configPath) {
		manifestBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read package.json: %v", err)
		}

		var packageConfig map[string]json.RawMessage
		if err := json.Unmarshal(manifestBytes, &packageConfig); err != nil {
			return nil, fmt.Errorf("could not unmarshal package.json: %v", err)
		}

		manifestBytes, ok := packageConfig["smallweb"]
		if !ok {
			return &config, nil
		}

		if err := json.Unmarshal(manifestBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal package.json: %v", err)
		}

		return &config, nil
	}

	return &config, nil
}

func LoadEnv(dir string) (map[string]string, error) {
	env := os.Environ()
	envMap := make(map[string]string)
	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		envMap[pair[0]] = pair[1]
	}

	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("could not get executable path: %v", err)
	}
	envMap["SMALLWEB_EXEC_PATH"] = executable

	dotenv, err := godotenv.Read(filepath.Join(dir, ".env"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return envMap, nil
		}

		return nil, fmt.Errorf("could not read .env: %v", err)
	}

	for key, value := range dotenv {
		envMap[key] = value
	}

	return envMap, nil
}

func (me *Worker) Entrypoint() (string, error) {
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

//go:embed deno/fetch.ts
var fetchBytes []byte

func (me *Worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	entrypoint, err := me.Entrypoint()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	freeport, err := GetFreePort()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tempfile, err := os.CreateTemp("", "sandbox-*.ts")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer os.Remove(tempfile.Name())
	if _, err := tempfile.Write(fetchBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	args := []string{"run"}
	args = append(args, me.Flags()...)

	url := fmt.Sprintf("https://%s%s", r.Host, r.URL.String())
	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
		"port":       freeport,
		"entrypoint": entrypoint,
		"url":        url,
	})
	args = append(args, tempfile.Name(), input.String())

	deno, err := DenoExecutable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(deno, args...)
	cmd.Dir = me.Root()

	var env []string
	for k, v := range me.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("Failed to send interrupt signal: %v", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-time.After(5 * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill proces %v", err)
			}
			return
		case <-done:
			return
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Scan()
	line := scanner.Text()
	if !(line == "READY") {
		http.Error(w, "could not start server", http.StatusInternalServerError)
		return
	}

	go func() {
		for scanner.Scan() {
			os.Stdout.WriteString(scanner.Text() + "\n")
		}
	}()

	// handle websockets
	if r.Header.Get("Upgrade") == "websocket" {
		serverConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading connection: %v", err)
			return
		}
		defer serverConn.Close()

		clientConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d%s", freeport, r.URL.Path), nil)
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

	request, err := http.NewRequest(r.Method, fmt.Sprintf("http://localhost:%d%s", freeport, r.URL.String()), r.Body)
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
			if err == io.EOF {
				break
			}
			return
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

//go:embed deno/run.ts
var runBytes []byte

func (me *Worker) Run(args []string) error {
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
	if _, err := tempfile.Write(runBytes); err != nil {
		return err
	}

	denoArgs := []string{"run"}
	denoArgs = append(denoArgs, me.Flags()...)

	input := strings.Builder{}
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]any{
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
