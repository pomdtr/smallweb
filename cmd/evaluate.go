package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/google/shlex"
)

var extensions = []string{".js", ".ts", ".jsx", ".tsx"}

type CommandInput struct {
	Req        *SerializedRequest `json:"req"`
	Entrypoint string             `json:"entrypoint"`
	Env        map[string]string  `json:"env"`
	Output     string             `json:"output"`
}

func inferEntrypoint(dir string) (string, error) {
	for _, ext := range extensions {
		entrypoint := path.Join(dir, "mod"+ext)
		if exists(entrypoint) {
			return entrypoint, nil
		}
	}

	entrypoint := path.Join(dir, "index.html")
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

	defaultFlags := []string{"--allow-net", "--allow-read=.", "allow-write=.", "--env"}
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
	outputFile := path.Join(tempdir, "response.json")

	input := CommandInput{
		Req:        req,
		Entrypoint: entrypoint,
		Output:     outputFile,
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

	cmd := exec.Command(deno, args...)
	cmd.Dir = rootDir
	stdin := bytes.Buffer{}
	encoder := json.NewEncoder(&stdin)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(input); err != nil {
		return nil, err
	}

	cmd.Stdin = &stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	f, err := os.Open(outputFile)
	if err != nil {
		return nil, err
	}

	var res SerializedResponse
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
}
