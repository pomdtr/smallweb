package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
	"github.com/pomdtr/smallweb/worker"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml ./openapi.json

type Server struct {
	domain string
}

func NewHandler(domain string) http.Handler {
	server := &Server{domain: domain}
	handler := Handler(server)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" {
			spec, err := GetSwagger()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			w.Header().Set("Content-Type", "application/json")
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			encoder.Encode(spec)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// GetV0AppsAppEnv implements ServerInterface.
func (me *Server) GetApp(w http.ResponseWriter, r *http.Request, appname string) {
	a, err := app.LoadApp(filepath.Join(utils.RootDir(), appname), me.domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	manifestPath := filepath.Join(a.Root(), "manifest.json")
	if !utils.FileExists(manifestPath) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(App{
			Name: a.Name,
			Url:  a.URL,
		})
		return
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(App{
			Name: a.Name,
			Url:  a.URL,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(App{
		Name:     a.Name,
		Url:      a.URL,
		Manifest: &manifest,
	})
}

func (me *Server) GetApps(w http.ResponseWriter, r *http.Request) {
	names, err := app.ListApps(utils.RootDir())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apps []App
	for _, name := range names {
		a, err := app.LoadApp(filepath.Join(utils.RootDir(), name), me.domain)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		manifestPath := filepath.Join(a.Root(), "manifest.json")
		if !utils.FileExists(manifestPath) {
			apps = append(apps, App{
				Name: a.Name,
				Url:  a.URL,
			})
			continue
		}

		manifestBytes, err := os.ReadFile(manifestPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var manifest map[string]interface{}
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			continue
		}

		apps = append(apps, App{
			Name:     a.Name,
			Url:      a.URL,
			Manifest: &manifest,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type CommandOutput struct {
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
	Code    int    `json:"code"`
	Success bool   `json:"success"`
}

func (me *Server) PostV0RunApp(w http.ResponseWriter, r *http.Request, appname string) {
	a, err := app.LoadApp(filepath.Join(utils.RootDir(), appname), me.domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var body PostV0RunAppJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	wk := worker.NewWorker(a)
	command, err := wk.Command(body.Args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if body.Stdin != nil {
		stdin, err := base64.StdEncoding.DecodeString(*body.Stdin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		command.Stdin = bytes.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var code int
	if err := command.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		code = exitErr.ExitCode()
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(CommandOutput{
		Stdout:  base64.StdEncoding.EncodeToString(stdout.Bytes()),
		Stderr:  base64.StdEncoding.EncodeToString(stderr.Bytes()),
		Code:    code,
		Success: code == 0,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
