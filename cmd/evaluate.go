package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
	"github.com/joho/godotenv"
)

var extensions = []string{".js", ".ts", ".jsx", ".tsx"}
var dataHome = path.Join(xdg.DataHome, "smallweb")
var sandboxPath = path.Join(dataHome, "sandbox.ts")

//go:embed deno/sandbox.ts
var sandboxBytes []byte

type CommandInput struct {
	Req        Request           `json:"req"`
	Entrypoint string            `json:"entrypoint"`
	Env        map[string]string `json:"env"`
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

func inferEntrypoint(rootDir string) (string, error) {
	for _, ext := range extensions {
		entrypoint := path.Join(rootDir, "mod"+ext)
		if exists(entrypoint) {
			return entrypoint, nil
		}
	}

	entrypoint := path.Join(rootDir, "index.html")
	if exists(entrypoint) {
		return entrypoint, nil
	}

	return "", fmt.Errorf("entrypoint not found")
}

type Request struct {
	Url     string     `json:"url"`
	Method  string     `json:"method"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body,omitempty"`
}

func (r Request) Username() (string, error) {
	url, err := url.Parse(r.Url)
	if err != nil {
		return "", err
	}

	subdomain := strings.Split(url.Host, ".")[0]
	parts := strings.Split(subdomain, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid subdomain")
	}

	return parts[0], nil
}

func (r Request) Alias() (string, error) {
	url, err := url.Parse(r.Url)
	if err != nil {
		return "", err
	}

	subdomain := strings.Split(url.Host, ".")[0]
	parts := strings.SplitN(subdomain, "-", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid subdomain")
	}

	return parts[1], nil
}

type Response struct {
	Code    int        `json:"code"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body"`
}

type DenoClient struct {
	entrypoint string
}

func NewDenoClient(entrypoint string) *DenoClient {
	return &DenoClient{entrypoint: entrypoint}
}

func (me *DenoClient) Do(req *Request) (*Response, error) {
	rootDir := path.Dir(me.entrypoint)
	if strings.HasSuffix(me.entrypoint, ".html") {
		fileServer := http.FileServer(http.Dir(rootDir))
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(req.Method, req.Url, nil)
		fileServer.ServeHTTP(rr, req)

		var headers [][]string
		for key, values := range rr.Result().Header {
			headers = append(headers, []string{key, values[0]})
		}

		body, err := io.ReadAll(rr.Result().Body)
		if err != nil {
			return nil, err
		}

		return &Response{
			Code:    rr.Result().StatusCode,
			Headers: headers,
			Body:    body,
		}, nil

	}

	deno, err := denoExecutable()
	if err != nil {
		return nil, err
	}

	freeport, err := GetFreePort()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", freeport))
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(deno, "run", "--allow-all", "--unstable-kv", sandboxPath, strconv.Itoa(freeport))
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	var env map[string]string
	if exists(".env") {
		envMap, err := godotenv.Read(".env")
		if err != nil {
			return nil, err
		}

		env = envMap
	}

	go cmd.Run()

	conn, err := ln.Accept()
	if err != nil {
		return nil, err
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(&CommandInput{
		Req:        *req,
		Entrypoint: me.entrypoint,
		Env:        env,
	}); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(conn)
	var res Response
	if err := decoder.Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
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
