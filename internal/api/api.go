package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
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
	router := http.NewServeMux()
	api := humago.New(router, huma.DefaultConfig("My API", "1.0.0"))

	blobsDir := filepath.Join(conf.String("dir"), ".smallweb", "data", "blobs")

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/v1/blobs",
		Description: "List Blobs",
	}, func(ctx context.Context, input *struct {
		Prefix string `query:"prefix" doc:"Filter blobs by prefix"`
	}) (*GetBlobsOutput, error) {
		// walk the blobs directory and list all blobs
		matches, err := filepath.Glob(filepath.Join(blobsDir, "*"))
		if err != nil {
			return nil, err
		}

		var keys []string
		prefix := ""
		if input != nil {
			prefix = input.Prefix
		}

		for _, p := range matches {
			name := filepath.Base(p)
			if prefix != "" {
				if !strings.HasPrefix(name, prefix) {
					continue
				}
			}
			keys = append(keys, name)
		}

		return &GetBlobsOutput{Body: keys}, nil
	})

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/v1/blobs/{key}",
		Description: "Retrieve a blob by its key",
		OperationID: "getBlob",
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Blob Content",
				Content: map[string]*huma.MediaType{
					"application/octet-stream": {},
				},
			},
		},
	}, func(ctx context.Context, input *struct {
		Key string `path:"key" doc:"The blob key"`
	}) (*GetBlobOutput, error) {
		fp := filepath.Join(blobsDir, input.Key)
		content, err := os.ReadFile(fp)
		if err != nil {
			return nil, err
		}

		return &GetBlobOutput{
			ContentType: "application/octet-stream",
			Body:        content,
		}, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
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
		Method: http.MethodGet,
		Path:   "/v1/apps/{app}",
	}, func(ctx context.Context, i *struct {
	}) (*GetAppOutput, error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Path:   "/v1/sqlite/query",
	}, func(ctx context.Context, i *struct{}) (o *struct{}, err error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Path:   "/v1/sqlite/batch",
	}, func(ctx context.Context, i *struct{}) (o *struct{}, err error) {
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Path:   "/v1/email",
	}, func(ctx context.Context, i *struct{}) (o *struct{}, err error) {
		return nil, nil
	})

	return router
}
