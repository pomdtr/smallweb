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
	"github.com/pomdtr/smallweb/server"
)

var EXTENSIONS = []string{".js", ".ts", ".jsx", ".tsx"}
var dataHome = path.Join(xdg.DataHome, "smallweb")
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

	// refresh sandbox code
	if err := os.WriteFile(sandboxPath, sandboxBytes, 0644); err != nil {
		log.Fatal(err)
	}
}

func inferEntrypoints(name string) (*WorkerEntrypoints, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &WorkerEntrypoints{
		Http: func() string {
			for _, ext := range EXTENSIONS {
				entrypoint := path.Join(homedir, "www", name, "http"+ext)
				if exists(entrypoint) {
					return entrypoint
				}
			}

			entrypoint := path.Join(homedir, "www", name, "index.html")
			if exists(entrypoint) {
				return entrypoint
			}
			return ""
		}(),
		Cli: func() string {
			for _, ext := range EXTENSIONS {
				entrypoint := path.Join(homedir, "www", name, "cli"+ext)
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

type Handler struct {
	entrypoints WorkerEntrypoints
}

func NewHandler(alias string) (*Handler, error) {
	entrypoints, err := inferEntrypoints(alias)
	if err != nil {
		return nil, fmt.Errorf("could not infer entrypoint: %v", err)
	}

	return &Handler{entrypoints: *entrypoints}, nil
}

func (me *Handler) Cmd(args ...string) (*exec.Cmd, error) {
	deno, err := DenoExecutable()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(deno, args...)
	return cmd, nil
}

func (me *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if me.entrypoints.Http == "" {
		http.NotFound(rw, req)
		return
	}

	rootDir := path.Dir(me.entrypoints.Http)
	if strings.HasSuffix(me.entrypoints.Http, ".html") {
		fileServer := http.FileServer(http.Dir(rootDir))
		fileServer.ServeHTTP(rw, req)
		return
	}

	freeport, err := server.GetFreePort()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd, err := me.Cmd("run", "--allow-read=.", "--allow-write=.", "--allow-net", "--allow-env", sandboxPath, me.entrypoints.Http, strconv.Itoa(freeport))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "SMALLWEB_PORT="+strconv.Itoa(freeport))
	cmd.Env = append(cmd.Env, "SMALLWEB_ENTRYPOINT="+filepath.Base(me.entrypoints.Http))
	cmd.Dir = rootDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Scan()
	line := scanner.Text()
	if !(line == "READY") {
		http.Error(rw, "could not start server", http.StatusInternalServerError)
		return
	}

	request, err := http.NewRequest(req.Method, fmt.Sprintf("http://127.0.0.1:%d%s", freeport, req.URL.String()), req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range req.Header {
		for _, vv := range v {
			request.Header.Add(k, vv)
		}
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			rw.Header().Add(k, vv)
		}
	}

	rw.WriteHeader(resp.StatusCode)

	flusher := rw.(http.Flusher)
	// Stream the response body to the client
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := rw.Write(buf[:n])
			if writeErr != nil {
				return
			}
			flusher.Flush() // flush the buffer to the client
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (me *Handler) Run(runArgs []string) error {
	if me.entrypoints.Cli == "" {
		return fmt.Errorf("entrypoint not found")
	}

	args := []string{"run", "--allow-all", me.entrypoints.Cli}
	args = append(args, runArgs...)

	command, err := me.Cmd(args...)
	if err != nil {
		return err
	}

	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

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
