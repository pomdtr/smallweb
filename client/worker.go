package client

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/pomdtr/smallweb/proxy"
)

var EXTENSIONS = []string{".js", ".ts", ".jsx", ".tsx"}
var SMALLWEB_ROOT string
var SmallwebDir string

var dataHome = path.Join(xdg.DataHome, "smallweb")
var cacheHome = path.Join(xdg.CacheHome, "smallweb")
var sandboxPath = path.Join(dataHome, "sandbox.ts")

//go:embed deno/sandbox.ts
var sandboxBytes []byte

type FetchInput struct {
	Entrypoint string     `json:"entrypoint"`
	Port       int        `json:"port"`
	Url        string     `json:"url"`
	Headers    [][]string `json:"headers"`
	Method     string     `json:"method"`
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	if err := os.MkdirAll(dataHome, 0755); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(cacheHome, 0755); err != nil {
		log.Fatal(err)
	}

	// refresh sandbox code
	if err := os.WriteFile(sandboxPath, sandboxBytes, 0644); err != nil {
		log.Fatal(err)
	}

	if env, ok := os.LookupEnv("SMALLWEB_ROOT"); ok {
		SMALLWEB_ROOT = env
	} else if home, err := os.UserHomeDir(); err == nil {
		SMALLWEB_ROOT = path.Join(home, "www")
	} else {
		log.Fatal(fmt.Errorf("could not determine smallweb root, please set SMALLWEB_ROOT"))
	}

	SmallwebDir = path.Join(SMALLWEB_ROOT, ".smallweb")
	if err := os.MkdirAll(filepath.Join(SmallwebDir, "logs"), 0755); err != nil {
		log.Fatalf("could not create logs directory: %v", err)
	}

}

func inferEntrypoints(name string) (*WorkerEntrypoints, error) {

	return &WorkerEntrypoints{
		Http: func() string {
			for _, ext := range EXTENSIONS {
				entrypoint := path.Join(SMALLWEB_ROOT, name, "main"+ext)
				if exists(entrypoint) {
					return entrypoint
				}
			}

			entrypoint := path.Join(SMALLWEB_ROOT, name, "index.html")
			if exists(entrypoint) {
				return entrypoint
			}
			return ""
		}(),
		Cli: func() string {
			for _, ext := range EXTENSIONS {
				entrypoint := path.Join(SMALLWEB_ROOT, name, "cli"+ext)
				if exists(entrypoint) {
					return entrypoint
				}
			}
			return ""
		}(),
	}, nil
}

type WorkerEntrypoints struct {
	Http string
	Cli  string
}

type Worker struct {
	alias       string
	entrypoints WorkerEntrypoints
}

func NewWorker(alias string) (*Worker, error) {
	entrypoints, err := inferEntrypoints(alias)
	if err != nil {
		return nil, fmt.Errorf("could not infer entrypoint: %v", err)
	}

	return &Worker{alias: alias, entrypoints: *entrypoints}, nil
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

func (me *Worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if me.entrypoints.Http == "" {
		http.NotFound(w, r)
		return
	}
	rootDir := path.Dir(me.entrypoints.Http)

	if strings.HasSuffix(me.entrypoints.Http, ".html") {
		fileServer := http.FileServer(http.Dir(rootDir))
		fileServer.ServeHTTP(w, r)
		return
	}

	freeport, err := proxy.GetFreePort()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd, err := me.Cmd("run", "--allow-all", sandboxPath, "--entrypoint", me.entrypoints.Http, "--port", strconv.Itoa(freeport))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd.Dir = rootDir
	if exists(filepath.Join(rootDir, ".env")) {
		envMap, err := godotenv.Read(filepath.Join(rootDir, ".env"))
		if err != nil {
			http.Error(w, fmt.Sprintf("could not read .env file: %v", err), http.StatusInternalServerError)
			return
		}

		cmd.Env = os.Environ()
		for key, value := range envMap {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	logFile, err := os.OpenFile(filepath.Join(SmallwebDir, "logs", me.alias+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer logFile.Close()

	go func() {
		for scanner.Scan() {
			logFile.WriteString(scanner.Text() + "\n")
			if err := scanner.Err(); err != nil {
				break
			}
		}
	}()

	go func() {
		io.Copy(logFile, stderr)
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

	request, err := http.NewRequest(r.Method, fmt.Sprintf("http://127.0.0.1:%d%s", freeport, r.URL.String()), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range r.Header {
		for _, vv := range v {
			request.Header.Add(k, vv)
		}
	}

	resp, err := http.DefaultClient.Do(request)
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
	if me.entrypoints.Cli == "" {
		return fmt.Errorf("entrypoint not found")
	}
	rootDir := filepath.Dir(me.entrypoints.Cli)

	args := []string{"run", "--allow-all", me.entrypoints.Cli}
	args = append(args, runArgs...)

	command, err := me.Cmd(args...)
	if err != nil {
		return err
	}
	command.Dir = rootDir
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if exists(filepath.Join(rootDir, ".env")) {
		envMap, err := godotenv.Read(filepath.Join(rootDir, ".env"))
		if err != nil {
			return fmt.Errorf("could not read .env file: %v", err)
		}

		command.Env = os.Environ()
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

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if denoPath, err := exec.LookPath("deno"); err == nil {
		return denoPath, nil
	}

	denoPath := filepath.Join(homedir, ".deno", "bin", "deno")
	if exists(denoPath) {
		return denoPath, nil
	}

	return "", fmt.Errorf("deno executable not found")
}
