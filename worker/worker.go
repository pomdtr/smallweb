package worker

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/adrg/xdg"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/tailscale/hujson"
)

type Config struct {
	Permissions Permissions `json:"permissions"`
}

type Permission struct {
	All   bool     `json:"all"`
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type Permissions struct {
	All   bool       `json:"all"`
	Read  Permission `json:"read"`
	Write Permission `json:"write"`
	Net   Permission `json:"net"`
	Env   Permission `json:"env"`
	Run   Permission `json:"run"`
	Sys   Permission `json:"sys"`
	Ffi   Permission `json:"ffi"`
}

func (p *Permission) UnmarshalJSON(data []byte) error {
	var all bool
	if err := json.Unmarshal(data, &all); err == nil {
		p.All = all
		return nil
	}

	var allow []string
	if err := json.Unmarshal(data, &allow); err == nil {
		p.Allow = allow
		return nil
	}

	var object map[string][]string
	if err := json.Unmarshal(data, &object); err == nil {
		if allow, ok := object["allow"]; ok {
			p.Allow = allow
		}
		if deny, ok := object["deny"]; ok {
			p.Deny = deny
		}
		return nil
	}

	return fmt.Errorf("could not unmarshal permission")
}

func (p *Permissions) Flags(rootDir string) []string {
	if p.All {
		return []string{"--allow-all"}
	}

	flags := []string{}

	if p.Read.All {
		flags = append(flags, "--allow-read")
	} else {
		if len(p.Read.Allow) > 0 {
			var allow []string
			for _, path := range p.Read.Allow {
				if filepath.IsAbs(path) {
					allow = append(allow, path)
					continue
				}

				allow = append(allow, filepath.Join(rootDir, path))
			}

			flags = append(flags, "--allow-read="+strings.Join(allow, ","))
		}
		if len(p.Read.Deny) > 0 {
			var deny []string
			for _, path := range p.Read.Deny {
				if filepath.IsAbs(path) {
					deny = append(deny, path)
					continue
				}

				deny = append(deny, filepath.Join(rootDir, path))
			}

			flags = append(flags, "--deny-read="+strings.Join(deny, ","))
		}
	}

	if p.Write.All {
		flags = append(flags, "--allow-write")
	} else {
		if len(p.Write.Allow) > 0 {
			var allow []string
			for _, path := range p.Write.Allow {
				if filepath.IsAbs(path) {
					allow = append(allow, path)
					continue
				}

				allow = append(allow, filepath.Join(rootDir, path))
			}

			flags = append(flags, "--allow-write="+strings.Join(allow, ","))
		}
		if len(p.Write.Deny) > 0 {
			var deny []string
			for _, path := range p.Write.Deny {
				if filepath.IsAbs(path) {
					deny = append(deny, path)
					continue
				}

				deny = append(deny, filepath.Join(rootDir, path))
			}

			flags = append(flags, "--deny-write="+strings.Join(deny, ","))
		}
	}

	if p.Net.All {
		flags = append(flags, "--allow-net")
	} else {
		if len(p.Net.Allow) > 0 {
			flags = append(flags, "--allow-net="+strings.Join(p.Net.Allow, ","))
		}
		if len(p.Net.Deny) > 0 {
			flags = append(flags, "--deny-net="+strings.Join(p.Net.Deny, ","))
		}
	}

	if p.Env.All {
		flags = append(flags, "--allow-env")
	} else {
		if len(p.Env.Allow) > 0 {
			flags = append(flags, "--allow-env="+strings.Join(p.Env.Allow, ","))
		}
		if len(p.Env.Deny) > 0 {
			flags = append(flags, "--deny-env="+strings.Join(p.Env.Deny, ","))
		}
	}

	if p.Run.All {
		flags = append(flags, "--allow-run")
	} else {
		if len(p.Run.Allow) > 0 {
			flags = append(flags, "--allow-run="+strings.Join(p.Run.Allow, ","))
		}
		if len(p.Run.Deny) > 0 {
			flags = append(flags, "--deny-run="+strings.Join(p.Run.Deny, ","))
		}
	}

	if p.Sys.All {
		flags = append(flags, "--allow-sys")
	} else {
		if len(p.Sys.Allow) > 0 {
			flags = append(flags, "--allow-sys="+strings.Join(p.Sys.Allow, ","))
		}
		if len(p.Sys.Deny) > 0 {
			flags = append(flags, "--deny-sys="+strings.Join(p.Sys.Deny, ","))
		}
	}

	if p.Ffi.All {
		flags = append(flags, "--allow-ffi")
	} else {
		if len(p.Ffi.Allow) > 0 {
			flags = append(flags, "--allow-ffi="+strings.Join(p.Ffi.Allow, ","))
		}
		if len(p.Ffi.Deny) > 0 {
			flags = append(flags, "--deny-ffi="+strings.Join(p.Ffi.Deny, ","))
		}
	}

	return flags
}

var EXTENSIONS = []string{".js", ".ts", ".jsx", ".tsx"}
var SMALLWEB_ROOT string
var SmallwebDir string

var dataHome = path.Join(xdg.DataHome, "smallweb")

//go:embed sandbox.ts
var sandboxBytes []byte
var sandboxTemplate = template.Must(template.New("sandbox").Parse(string(sandboxBytes)))

type FetchInput struct {
	Entrypoint string     `json:"entrypoint"`
	Port       int        `json:"port"`
	Url        string     `json:"url"`
	Headers    [][]string `json:"headers"`
	Method     string     `json:"method"`
}

func FileExists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

func init() {
	if err := os.MkdirAll(dataHome, 0755); err != nil {
		log.Fatal(err)
	}

	if env, ok := os.LookupEnv("SMALLWEB_ROOT"); ok {
		SMALLWEB_ROOT = env
	} else if home, err := os.UserHomeDir(); err == nil {
		SMALLWEB_ROOT = path.Join(home, "www")
	} else {
		log.Fatal(fmt.Errorf("could not determine smallweb root, please set SMALLWEB_ROOT"))
	}

	// ensure smallweb is in the path
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("could not determine executable path: %v", err)
	}
	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(executable), os.Getenv("PATH")))
}

type WorkerEntrypoints struct {
	Http string
	Cli  string
}

type Worker struct {
	alias string
}

func NewWorker(alias string) (*Worker, error) {

	return &Worker{alias: alias}, nil
}

func (me *Worker) Cmd(args ...string) (*exec.Cmd, error) {
	deno, err := DenoExecutable()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(deno, args...)
	return cmd, nil
}

var upgrader = websocket.Upgrader{} // use default options

func (me *Worker) LoadConfig() (*Config, error) {
	var appDir = filepath.Join(SMALLWEB_ROOT, me.alias)
	defaultConfig := Config{
		Permissions: Permissions{
			Read: Permission{
				Allow: []string{appDir},
			},
			Write: Permission{
				Allow: []string{appDir},
				Deny: []string{
					filepath.Join(appDir, "smallweb.json"),
					filepath.Join(appDir, "smallweb.jsonc"),
					filepath.Join(appDir, "deno.json"),
					filepath.Join(appDir, "deno.jsonc"),
				},
			},
			Net: Permission{
				All: true,
			},
			Env: Permission{
				All: true,
			},
		},
	}

	if configPath := filepath.Join(SMALLWEB_ROOT, me.alias, "smallweb.json"); FileExists(configPath) {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		var config Config
		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(SMALLWEB_ROOT, me.alias, "smallweb.jsonc"); FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read deno.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return nil, fmt.Errorf("could not standardize deno.jsonc: %v", err)
		}

		var config Config
		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(SMALLWEB_ROOT, me.alias, "deno.json"); FileExists(configPath) {
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
			return &defaultConfig, nil
		}

		var config Config
		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(SMALLWEB_ROOT, me.alias, "deno.jsonc"); FileExists(configPath) {
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
			return &defaultConfig, nil
		}

		var config Config
		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	return &defaultConfig, nil
}

func (me *Worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var appDir = filepath.Join(SMALLWEB_ROOT, me.alias)

	if !FileExists(filepath.Join(SMALLWEB_ROOT, me.alias)) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var entrypoint string
	for _, extension := range EXTENSIONS {
		var candidate = filepath.Join(appDir, "main"+extension)
		if FileExists(candidate) {
			entrypoint = candidate
			break
		}
	}

	if entrypoint == "" {
		if FileExists(filepath.Join(appDir, "dist", "index.html")) {
			fileServer := http.FileServer(http.Dir(filepath.Join(appDir, "dist")))
			fileServer.ServeHTTP(w, r)
			return
		}

		fileServer := http.FileServer(http.Dir(appDir))
		fileServer.ServeHTTP(w, r)
		return
	}

	freeport, err := GetFreePort()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, err := me.LoadConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !config.Permissions.All && !config.Permissions.Net.All {
		config.Permissions.Net.Allow = append(config.Permissions.Net.Allow, fmt.Sprintf("0.0.0.0:%d", freeport))
	}

	host := r.Host
	protocol := r.Header.Get("X-Forwarded-Proto")
	if protocol == "" {
		protocol = "http"
	}

	var stdin bytes.Buffer

	url := fmt.Sprintf("%s://%s%s", protocol, host, r.URL.String())
	if err := sandboxTemplate.Execute(&stdin, map[string]interface{}{
		"Port":       freeport,
		"ModURL":     entrypoint,
		"RequestURL": url,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	args := []string{"run", "--unstable-kv", "--unstable-temporal"}
	flags := config.Permissions.Flags(appDir)
	args = append(args, flags...)
	args = append(args, "-")

	cmd, err := me.Cmd(args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd.Dir = appDir
	cmd.Stdin = &stdin
	cmd.Env = os.Environ()

	if FileExists(filepath.Join(SMALLWEB_ROOT, ".env")) {
		envMap, err := godotenv.Read(filepath.Join(SMALLWEB_ROOT, ".env"))
		if err != nil {
			http.Error(w, fmt.Sprintf("could not read .env file: %v", err), http.StatusInternalServerError)
			return
		}

		for key, value := range envMap {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

	if FileExists(filepath.Join(appDir, ".env")) {
		envMap, err := godotenv.Read(filepath.Join(appDir, ".env"))
		if err != nil {
			http.Error(w, fmt.Sprintf("could not read .env file: %v", err), http.StatusInternalServerError)
			return
		}

		for key, value := range envMap {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

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
	defer cmd.Process.Kill()

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
				log.Printf("Error writing response: %v", writeErr)
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

func (me *Worker) Run(runArgs []string) error {
	appDir := filepath.Join(SMALLWEB_ROOT, me.alias)

	var entrypoint string
	for _, extension := range EXTENSIONS {
		candidate := filepath.Join(appDir, "cli"+extension)
		if FileExists(candidate) {
			entrypoint = candidate
			break
		}
	}

	args := []string{"run", "--allow-all", entrypoint}
	args = append(args, runArgs...)

	command, err := me.Cmd(args...)
	if err != nil {
		return err
	}
	command.Dir = appDir
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	command.Env = os.Environ()
	if FileExists(filepath.Join(SMALLWEB_ROOT, ".env")) {
		envMap, err := godotenv.Read(filepath.Join(SMALLWEB_ROOT, ".env"))
		if err != nil {
			return fmt.Errorf("could not read .env file: %v", err)
		}

		for key, value := range envMap {
			command.Env = append(command.Env, key+"="+value)
		}
	}

	if FileExists(filepath.Join(appDir, ".env")) {
		envMap, err := godotenv.Read(filepath.Join(appDir, ".env"))
		if err != nil {
			return fmt.Errorf("could not read .env file: %v", err)
		}

		for key, value := range envMap {
			command.Env = append(command.Env, key+"="+value)
		}
	}

	return command.Run()
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
		if FileExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("deno executable not found")
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
