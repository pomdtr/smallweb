package cmd

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
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
	Req        *SerializedRequest `json:"req"`
	Entrypoint string             `json:"entrypoint"`
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

func inferEntrypoint(rootDir string, name string) (string, error) {
	for _, ext := range extensions {
		entrypoint := path.Join(rootDir, name, "mod"+ext)
		if exists(entrypoint) {
			return entrypoint, nil
		}
	}

	entrypoint := path.Join(rootDir, name, "index.html")
	if exists(entrypoint) {
		return entrypoint, nil
	}

	return "", fmt.Errorf("entrypoint not found")
}

type SerializedRequest struct {
	Url     string     `json:"url"`
	Method  string     `json:"method"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body,omitempty"`
}

type SerializedResponse struct {
	Code    int        `json:"code"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body"`
}

func Evaluate(entrypoint string, req *SerializedRequest) (*SerializedResponse, error) {
	rootDir := path.Dir(entrypoint)
	if strings.HasSuffix(entrypoint, ".html") {
		fileServer := http.FileServer(http.Dir(rootDir))
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(req.Method, req.Url, nil)
		fileServer.ServeHTTP(rr, req)

		var headers [][]string
		for key, values := range rr.Result().Header {
			headers = append(headers, []string{key, values[0]})
		}

		return &SerializedResponse{
			Code:    rr.Code,
			Headers: headers,
			Body:    rr.Body.Bytes(),
		}, nil

	}

	input := CommandInput{
		Req:        req,
		Entrypoint: entrypoint,
	}

	deno, err := denoExecutable()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(deno, "run", "--allow-net", "--allow-read=.", "--allow-write=./data", "--allow-env", "--unstable-kv", sandboxPath)
	cmd.Dir = rootDir
	if exists(path.Join(rootDir, ".env")) {
		envMap, err := godotenv.Read(".env")
		if err != nil {
			return nil, err
		}

		for key, value := range envMap {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	stdin := bytes.Buffer{}
	encoder := json.NewEncoder(&stdin)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(input); err != nil {
		return nil, err
	}

	// TODO: use websocket instead of stdin/stdout
	cmd.Stdin = &stdin
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s: %s", err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("%s: %s", err, output)
	}

	var res SerializedResponse
	if err := json.Unmarshal(output, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
