package worker

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
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
	"github.com/pomdtr/smallweb/utils"
	"github.com/tailscale/hujson"
)

type AppConfig struct {
	Serve       string      `json:"serve"`
	Crons       []CronJob   `json:"crons"`
	Permissions Permissions `json:"permissions"`
}

type CronJob struct {
	Path     string `json:"path"`
	Schedule string `json:"schedule"`
}

type CronJobRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
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

var dataHome = path.Join(xdg.DataHome, "smallweb")

//go:embed sandbox.ts
var sandboxBytes []byte
var sandboxTemplate = template.Must(template.New("sandbox").Parse(string(sandboxBytes)))

func init() {
	if err := os.MkdirAll(dataHome, 0755); err != nil {
		log.Fatal(err)
	}
}

type Worker struct {
	Dir string
	Env map[string]string
}

var upgrader = websocket.Upgrader{} // use default options

func (me *Worker) LoadConfig() (*AppConfig, error) {
	config := AppConfig{
		Permissions: Permissions{
			Read: Permission{
				Allow: []string{me.Dir},
			},
			Write: Permission{
				Allow: []string{me.Dir},
				Deny: []string{
					filepath.Join(me.Dir, "smallweb.json"),
					filepath.Join(me.Dir, "smallweb.jsonc"),
					filepath.Join(me.Dir, "deno.json"),
					filepath.Join(me.Dir, "deno.jsonc"),
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

	if configPath := filepath.Join(me.Dir, "smallweb.json"); utils.FileExists(configPath) {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return &config, nil
	}

	if configPath := filepath.Join(me.Dir, "smallweb.jsonc"); utils.FileExists(configPath) {
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

	if configPath := filepath.Join(me.Dir, "deno.json"); utils.FileExists(configPath) {
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

	if configPath := filepath.Join(me.Dir, "deno.jsonc"); utils.FileExists(configPath) {
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

	return &config, nil
}

func (me *Worker) LoadEnv() ([]string, error) {
	env := os.Environ()

	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("could not get executable path: %v", err)
	}
	env = append(env, fmt.Sprintf("SMALLWEB_EXEC_PATH=%s", executable))

	for key, value := range me.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	dotenv, err := godotenv.Read(filepath.Join(me.Dir, ".env"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return env, nil
		}

		return nil, fmt.Errorf("could not read .env: %v", err)
	}

	for key, value := range dotenv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env, nil
}

func (me *Worker) inferEntrypoint() (string, error) {
	for _, root := range []string{filepath.Join(me.Dir, "dist"), me.Dir} {
		for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
			path := filepath.Join(root, candidate)
			if utils.FileExists(path) {
				return path, nil
			}
		}
	}

	if utils.FileExists(filepath.Join(me.Dir, "dist", "index.html")) {
		return filepath.Join(me.Dir, "dist"), nil
	}

	return me.Dir, nil
}

type HTMLDir struct {
	dir http.Dir
}

func NewHtmlDir(dir http.Dir) HTMLDir {
	return HTMLDir{dir}
}

func (d HTMLDir) Open(name string) (http.File, error) {
	// Try name as supplied
	f, err := d.dir.Open(name)
	if !os.IsNotExist(err) {
		return f, err
	}

	// Not found, try with .html
	return d.dir.Open(name + ".html")
}

func (me *Worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !utils.FileExists(me.Dir) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	config, err := me.LoadConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	entrypoint := config.Serve
	if entrypoint != "" {
		entrypoint = filepath.Join(me.Dir, entrypoint)
	} else {
		e, err := me.inferEntrypoint()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entrypoint = e
	}

	info, err := os.Stat(entrypoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if info.IsDir() {
		server := http.FileServer(NewHtmlDir(http.Dir(entrypoint)))
		server.ServeHTTP(w, r)
		return
	}

	freeport, err := GetFreePort()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !config.Permissions.All && !config.Permissions.Net.All {
		config.Permissions.Net.Allow = append(config.Permissions.Net.Allow, fmt.Sprintf("0.0.0.0:%d", freeport))
	}

	url := fmt.Sprintf("https://%s%s", r.Host, r.URL.String())

	var stdin bytes.Buffer
	if err := sandboxTemplate.Execute(&stdin, map[string]interface{}{
		"Port":       freeport,
		"ModURL":     entrypoint,
		"RequestURL": url,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	args := []string{"run", "--unstable-kv", "--unstable-temporal"}
	flags := config.Permissions.Flags(me.Dir)
	args = append(args, flags...)
	args = append(args, "--location", fmt.Sprintf("https://%s/", r.Host))
	args = append(args, "-")

	env, err := me.LoadEnv()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deno, err := DenoExecutable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(deno, args...)
	cmd.Dir = filepath.Dir(entrypoint)
	cmd.Stdin = &stdin
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

func (me *Worker) Do(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	me.ServeHTTP(w, r)
	return w.Result(), nil
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
