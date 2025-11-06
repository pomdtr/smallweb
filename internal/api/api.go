package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/hostrouter"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/utils"
)

type GetBlobsOutput struct {
	Body []string
}

type GetBlobOutput struct {
	ContentType string `header:"Content-Type"`

	Body []byte
}

type GetAppOutput struct {
	Body struct {
	}
}

type GetAppsOutput struct {
	Body struct {
		Apps []string `json:"apps"`
	}
}

func NewHandler(conf *utils.Config) http.Handler {
	// Create a new router & API.
	r := chi.NewRouter()
	hr := hostrouter.New()

	hr.Map("api.localhost", apiRouter(conf))
	hr.Map("git.localhost", gitRouter(conf))
	hr.Map("*", catchAllRouter())

	r.Mount("/", hr)

	return r
}

func apiRouter(conf *utils.Config) chi.Router {
	r := chi.NewRouter()

	api := humachi.New(r, huma.DefaultConfig("Smallweb API", "v1.0.0"))

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps",
	}, func(ctx context.Context, i *struct {
	}) (*GetAppsOutput, error) {
		appList, err := app.List(conf.String("dir"))
		if err != nil {
			return nil, err
		}

		resp := &GetAppsOutput{}
		resp.Body.Apps = appList

		return resp, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps",
	}, func(ctx context.Context, i *struct{}) (*struct{}, error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps/{app}",
	}, func(ctx context.Context, i *struct {
	}) (*GetAppOutput, error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPut,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps/{app}",
	}, func(ctx context.Context, i *struct {
		App string `path:"app" doc:"The app name"`
	}) (*struct{}, error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodDelete,
		Path:   "/v1/apps/{app}",
		Tags:   []string{"Apps"},
	}, func(ctx context.Context, i *struct {
	}) (*struct{}, error) {
		return nil, nil
	})

	return r
}

func gitRouter(conf *utils.Config) chi.Router {
	r := chi.NewRouter()

	repoRoot := filepath.Join(conf.String("dir"))

	gitPath, err := exec.LookPath("git")
	if err != nil {
		r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "git binary not found on server", http.StatusInternalServerError)
		})
		return r
	}

	gitHandler := &cgi.Handler{
		Path: gitPath,
		Args: []string{"http-backend"},
		Env: []string{
			"GIT_PROJECT_ROOT=" + repoRoot,
			"GIT_HTTP_EXPORT_ALL=", // allow read-only access
		},
		Stderr: os.Stderr,
		InheritEnv: []string{
			"PATH", "HOME", // inherit some safe env vars
		},
	}

	r.HandleFunc("/{app}/*", func(w http.ResponseWriter, r *http.Request) {
		appname := chi.URLParam(r, "app")
		if conf.String(fmt.Sprintf("apps.%s.privacy", appname)) != "public" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/git-receive-pack") {
			http.Error(w, "pushes are disabled", http.StatusForbidden)
			return
		}

		fixChunked(r)

		gitHandler.ServeHTTP(w, r)
	})

	return r
}

func catchAllRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("unknown host: " + r.Host))
	})

	return r
}

func fixChunked(req *http.Request) {
	if len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == `chunked` {
		// hacking!
		req.TransferEncoding = nil
		req.Header.Set(`Transfer-Encoding`, `chunked`)

		// let cgi use Body.
		req.ContentLength = -1
	}
}
