package cmd

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

var dataHome = path.Join(xdg.DataHome, "smallweb")
var sandboxPath = path.Join(dataHome, "sandbox.ts")

//go:embed deno/sandbox.ts
var sandboxBytes []byte

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func denoExecutable() (string, error) {
	if env, ok := os.LookupEnv("DENO_EXEC_PATH"); ok {
		return env, nil
	}

	return exec.LookPath("deno")
}

var extensions = []string{".js", ".ts", ".jsx", ".tsx"}

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
	Status  int        `json:"status"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body"`
}

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use: "smallweb",
	}

	cmd.AddCommand(NewCmdServe())
	cmd.AddCommand(NewCmdTunnel())
	cmd.AddCommand(NewCmdProxy())
	cmd.InitDefaultCompletionCmd()

	return cmd
}

type CommandInput struct {
	Req        *SerializedRequest `json:"req"`
	Entrypoint string             `json:"entrypoint"`
	Env        map[string]string  `json:"env"`
	Output     string             `json:"output"`
}

func Evaluate(entrypoint string, req *SerializedRequest) (*SerializedResponse, error) {
	rootDir := path.Dir(entrypoint)
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

	cmd := exec.Command(deno, "run", "--env", "--allow-all", "--unstable-kv", sandboxPath)
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
