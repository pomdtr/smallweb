package cmd

import (
	"bufio"
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
	"github.com/google/shlex"
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
		entrypoint := path.Join(rootDir, name+ext)
		if exists(entrypoint) {
			return entrypoint, nil
		}
	}

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

func extractShebangFlags(script string) ([]string, error) {
	f, err := os.Open(script)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	defaultFlags := []string{"--allow-net", "--allow-read=.", "--allow-write=.", "--env"}
	scanner := bufio.NewScanner(f)
	if ok := scanner.Scan(); !ok {
		return defaultFlags, nil
	}

	line := scanner.Text()
	if !strings.HasPrefix(line, "#!") {
		return defaultFlags, nil
	}

	args, err := shlex.Split(line[2:])
	if err != nil {
		return nil, err
	}

	if args[0] != "/usr/bin/env" {
		return nil, fmt.Errorf("unsupported shebang")
	}
	args = args[1:]

	if args[0] != "-S" {
		return nil, fmt.Errorf("unsupported shebang")
	}
	args = args[1:]

	if args[0] != "deno" {
		return nil, fmt.Errorf("unsupported shebang")
	}
	args = args[1:]

	if args[0] != "serve" {
		return nil, fmt.Errorf("unsupported shebang")
	}
	args = args[1:]

	var flags []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--allow-") {
			flags = append(flags, arg)
		}

		if arg == "-A" {
			flags = append(flags, "--allow-all")
		}

		if strings.HasPrefix(arg, "--deny-") {
			flags = append(flags, arg)
		}

		if strings.HasPrefix(arg, "--unstable") {
			flags = append(flags, arg)
		}
	}

	return flags, nil
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

	tempdir, err := os.MkdirTemp("", "smallweb")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempdir)

	input := CommandInput{
		Req:        req,
		Entrypoint: entrypoint,
	}

	deno, err := denoExecutable()
	if err != nil {
		return nil, err
	}

	flags, err := extractShebangFlags(entrypoint)
	if err != nil {
		return nil, err
	}

	args := []string{"run"}
	args = append(args, flags...)
	args = append(args, sandboxPath)

	cmd := exec.Command(deno, args...)
	cmd.Dir = rootDir
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
