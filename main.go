package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var dataHome = path.Join(xdg.DataHome, "smallweb")
var sandboxPath = path.Join(dataHome, "sandbox.ts")

//go:embed embed/sandbox.ts
var sandboxBytes []byte

func exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func denoExecutable() (string, error) {
	if env, ok := os.LookupEnv("DENO_EXEC_PATH"); ok {
		return env, nil
	}

	return exec.LookPath("deno")
}

var extensions = []string{".js", ".ts", ".jsx", ".tsx"}

func inferEntrypoint(rootDir, alias string) (string, error) {
	for _, ext := range extensions {
		entrypoint := path.Join(rootDir, alias+ext)
		if exists, err := exists(entrypoint); err != nil {
			return "", err
		} else if exists {
			return entrypoint, nil
		}
	}

	for _, ext := range extensions {
		entrypoint := path.Join(rootDir, alias, "mod"+ext)
		if exists, err := exists(entrypoint); err != nil {
			return "", err
		} else if exists {
			return entrypoint, nil
		}
	}

	for _, ext := range extensions {
		entrypoint := path.Join(rootDir, alias, alias+ext)
		if exists, err := exists(entrypoint); err != nil {
			return "", err
		} else if exists {
			return entrypoint, nil
		}
	}

	entrypoint := path.Join(rootDir, alias, "index.html")
	if exists, err := exists(entrypoint); err != nil {
		return "", err
	} else if exists {
		return entrypoint, nil
	}

	return "", fmt.Errorf("entrypoint not found")
}

func loadEnv(root string, entrypoint string) (map[string]string, error) {
	if filepath.Ext(entrypoint) == ".html" {
		return make(map[string]string), nil
	}

	rootEnv := make(map[string]string)
	rootEnvPath := filepath.Join(root, ".env")
	if _, err := os.Stat(rootEnvPath); err == nil {
		rootEnv, err = godotenv.Read(rootEnvPath)
		if err != nil {
			return nil, err
		}
	}

	envPath := filepath.Join(filepath.Dir(entrypoint), ".env")
	if rootEnvPath == envPath {
		return rootEnv, nil
	}

	env, err := godotenv.Read(envPath)
	if err != nil {
		return nil, err
	}

	for k, v := range rootEnv {
		env[k] = v
	}

	return env, nil
}

func main() {
	cmd := NewCmdRoot()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type SerializedRequest struct {
	Url     string     `json:"url"`
	Method  string     `json:"method"`
	Headers [][]string `json:"headers"`
	Body    string     `json:"body,omitempty"`
}

func serializeRequestBody(body io.Reader) (string, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	res := strings.Builder{}
	if _, err := base64.NewEncoder(base64.StdEncoding, &res).Write(b); err != nil {
		return "", err
	}

	return res.String(), nil
}

func serializeRequest(req *http.Request) (SerializedRequest, error) {
	var res SerializedRequest

	url := req.URL
	url.Host = req.Host
	if req.TLS != nil {
		url.Scheme = "https"
	} else {
		url.Scheme = "http"
	}
	res.Url = url.String()

	res.Method = req.Method
	for k, v := range req.Header {
		res.Headers = append(res.Headers, []string{k, v[0]})
	}

	body, err := serializeRequestBody(req.Body)
	if err != nil {
		return res, err
	}
	res.Body = body

	return res, nil
}

type SerializedResponse struct {
	Status  int        `json:"status"`
	Headers [][]string `json:"headers"`
	Body    string     `json:"body"`
}

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use: "smallweb",
	}

	cmd.AddCommand(NewServeCmd())
	cmd.InitDefaultCompletionCmd()

	return cmd
}

func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "serve",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(dataHome, 0755); err != nil {
				return err
			}

			// refresh sandbox code
			if err := os.WriteFile(sandboxPath, sandboxBytes, 0644); err != nil {
				return err
			}

			rootDir := args[0]
			if exists, err := exists(rootDir); err != nil {
				return err
			} else if !exists {
				return fmt.Errorf("directory %s does not exist", rootDir)
			}

			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				return err
			}

			server := http.Server{
				Addr:    fmt.Sprintf(":%d", port),
				Handler: NewHandler(rootDir),
			}

			fmt.Fprintln(os.Stderr, "Listening on", server.Addr)
			return server.ListenAndServe()
		},
	}
	cmd.Flags().IntP("port", "p", 4321, "Port to listen on")
	return cmd
}

func NewHandler(rootDir string) http.Handler {
	return &Handler{rootDir: rootDir}

}

type Handler struct {
	rootDir string
}

type CommandInput struct {
	Req        SerializedRequest `json:"req"`
	Entrypoint string            `json:"entrypoint"`
	Env        map[string]string `json:"env"`
	Output     string            `json:"output"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	subdomain := strings.Split(host, ".")[0]

	entrypoint, err := inferEntrypoint(h.rootDir, subdomain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	env, err := loadEnv(h.rootDir, entrypoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := serializeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tempdir, err := os.MkdirTemp("", "smallweb")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempdir)
	output := path.Join(tempdir, "response.json")

	input := CommandInput{
		Req:        req,
		Entrypoint: entrypoint,
		Env:        env,
		Output:     output,
	}

	deno, err := denoExecutable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(deno, "run", "-A", sandboxPath)
	cmd.Dir = path.Dir(entrypoint)
	stdin := bytes.Buffer{}
	encoder := json.NewEncoder(&stdin)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cmd.Stdin = &stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f, err := os.Open(output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var res SerializedResponse
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(res.Body)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.Status != 200 {
		w.WriteHeader(res.Status)
	}
	for _, header := range res.Headers {
		w.Header().Set(header[0], header[1])
	}
	w.Write(body)

}
