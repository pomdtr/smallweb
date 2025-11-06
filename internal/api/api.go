package api

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/hostrouter"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/utils"
	"golang.org/x/net/webdav"
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

	hr.Map("webdav.localhost", webdavRouter(conf))
	hr.Map("api.localhost", apiRouter(conf))
	hr.Map("*", catchAllRouter())

	r.Mount("/", hr)

	return r
}

func webdavRouter(conf *utils.Config) chi.Router {
	r := chi.NewRouter()

	r.Handle("/*", &webdav.Handler{
		FileSystem: webdav.Dir(filepath.Join(conf.String("dir"), ".smallweb", "data")),
		LockSystem: webdav.NewMemLS(),
	})

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

func catchAllRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("unknown host: " + r.Host))
	})

	return r
}
