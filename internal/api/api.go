package api

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
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
	mux := chi.NewMux()
	mux.Handle("/webdav", &webdav.Handler{
		Prefix:     "/webdav/",
		FileSystem: webdav.Dir(filepath.Join(conf.String("dir"), ".smallweb", "data")),
		LockSystem: webdav.NewMemLS(),
	})

	mux.Route("/api", func(r chi.Router) {
		config := huma.DefaultConfig("Smallweb API", "v1.0.0")
		api := humachi.New(r, config)

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
	})

	return mux
}
