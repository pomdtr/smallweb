package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pomdtr/smallweb/app"
	"github.com/pomdtr/smallweb/utils"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml ./openapi.json

//go:embed template/*
var initTemplate embed.FS

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

func (me *Server) DeleteApp(w http.ResponseWriter, r *http.Request, appname string) {
	if err := os.RemoveAll(filepath.Join(utils.RootDir(), appname)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (me *Server) UpdateApp(w http.ResponseWriter, r *http.Request, appname string) {
	var body UpdateAppJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rootDir := utils.RootDir()
	src := filepath.Join(rootDir, appname)
	dst := filepath.Join(rootDir, body.Name)

	if _, err := os.Stat(src); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("app not found: %s", appname), http.StatusNotFound)
		return
	}

	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("app already exists: %s", body.Name), http.StatusConflict)
		return
	}

	if err := os.Rename(src, dst); err != nil {
		http.Error(w, fmt.Sprintf("failed to rename app: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (me *Server) CreateApp(w http.ResponseWriter, r *http.Request) {
	var body CreateAppJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rootDir := utils.RootDir()
	appDir := filepath.Join(rootDir, body.Name)
	if _, err := os.Stat(appDir); !os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("app already exists: %s", body.Name), http.StatusConflict)
	}

	subFs, err := fs.Sub(initTemplate, "template")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read template: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.CopyFS(appDir, subFs); err != nil {
		http.Error(w, fmt.Sprintf("failed to copy template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(App{
		Name: body.Name,
		Url:  fmt.Sprintf("https://%s.%s/", body.Name, me.domain),
	})
}
