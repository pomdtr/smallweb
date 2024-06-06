package cmd

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
)

var extensions = []string{".js", ".ts", ".jsx", ".tsx"}
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

type Email struct {
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Cc      string `json:"cc,omitempty"`
	Bcc     string `json:"bcc,omitempty"`
	Subject string `json:"subject,omitempty"`
	Text    string `json:"text,omitempty"`
	Html    string `json:"html,omitempty"`
}

func (me *Email) Username() (string, error) {
	host := strings.Split(me.To, "@")[0]
	parts := strings.Split(host, "-")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid subdomain")
	}

	return parts[len(parts)-1], nil
}

func (me *Email) App() (string, error) {
	host := strings.Split(me.To, "@")[0]
	parts := strings.Split(host, "-")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid subdomain")
	}

	return strings.Join(parts[:len(parts)-1], "-"), nil
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
	var lookupDirs []string
	if env, ok := os.LookupEnv("SMALLWEB_PATH"); ok {
		lookupDirs = strings.Split(env, ":")
	} else {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		lookupDirs = []string{path.Join(homedir, "www")}
	}

	return &WorkerEntrypoints{
		Http: func() string {
			for _, dir := range lookupDirs {
				for _, ext := range extensions {
					entrypoint := path.Join(dir, name, "http"+ext)
					if exists(entrypoint) {
						return entrypoint
					}
				}

				entrypoint := path.Join(dir, name, "index.html")
				if exists(entrypoint) {
					return entrypoint
				}
			}
			return ""
		}(),
		Cli: func() string {
			for _, dir := range lookupDirs {
				for _, ext := range extensions {
					entrypoint := path.Join(dir, name, "cli"+ext)
					if exists(entrypoint) {
						return entrypoint
					}
				}
			}
			return ""
		}(),
		Email: func() string {
			for _, dir := range lookupDirs {
				for _, ext := range extensions {
					entrypoint := path.Join(dir, name, "email"+ext)
					if exists(entrypoint) {
						return entrypoint
					}
				}
			}
			return ""
		}(),
	}, nil
}

type Request struct {
	Url     string     `json:"url"`
	Method  string     `json:"method"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func (r Request) Username() (string, error) {
	url, err := url.Parse(r.Url)
	if err != nil {
		return "", err
	}

	subdomain := strings.Split(url.Host, ".")[0]
	parts := strings.Split(subdomain, "-")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid subdomain")
	}

	return parts[len(parts)-1], nil
}

func (r Request) App() (string, error) {
	url, err := url.Parse(r.Url)
	if err != nil {
		return "", err
	}

	subdomain := strings.Split(url.Host, ".")[0]
	parts := strings.Split(subdomain, "-")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid subdomain")
	}

	return strings.Join(parts[:len(parts)-1], "-"), nil
}

type Response struct {
	Code    int        `json:"code"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body"`
}

type WorkerEntrypoints struct {
	Http  string
	Email string
	Cli   string
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
	deno, err := denoExecutable()
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

	freeport, err := GetFreePort()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd, err := me.Cmd("run", "--allow-all", "--unstable-kv", sandboxPath, me.entrypoints.Http, strconv.Itoa(freeport))
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

func (me *Handler) Email(email *Email) (*Email, error) {
	return nil, nil
	// if me.entrypoints.Email == "" {
	// 	return nil, fmt.Errorf("entrypoint not found")
	// }

	// rootDir := path.Dir(me.entrypoints.Email)
	// freeport, err := GetFreePort()
	// if err != nil {
	// 	return nil, err
	// }

	// ln, err := net.Listen("tcp", fmt.Sprintf(":%d", freeport))
	// if err != nil {
	// 	return nil, err
	// }

	// cmd, err := me.Cmd("run", "--allow-all", "--unstable-kv", sandboxPath, strconv.Itoa(freeport))
	// if err != nil {
	// 	return nil, err
	// }

	// cmd.Dir = rootDir
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// go cmd.Run()

	// conn, err := ln.Accept()
	// if err != nil {
	// 	return nil, err
	// }

	// encoder := json.NewEncoder(conn)
	// if err := encoder.Encode(&EmailInput{
	// 	Type:       "email",
	// 	Email:      *email,
	// 	Entrypoint: me.entrypoints.Email,
	// }); err != nil {
	// 	return nil, err
	// }

	// var msg Message
	// decoder := json.NewDecoder(conn)
	// if err := decoder.Decode(&msg); err != nil {
	// 	return nil, fmt.Errorf("could not decode message: %v", err)
	// }

	// switch msg.Type {
	// case "error":
	// 	b, err := json.Marshal(msg.Data)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	return nil, fmt.Errorf("error: %s", b)
	// default:
	// 	return nil, nil
	// }
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
