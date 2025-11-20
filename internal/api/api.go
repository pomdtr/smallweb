package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/luikyv/go-oidc/pkg/goidc"
	"github.com/luikyv/go-oidc/pkg/provider"
	"github.com/pomdtr/smallweb/internal/app"
	"github.com/pomdtr/smallweb/internal/utils"
	"golang.org/x/net/webdav"
)

type GetAppOutput struct {
	Body struct {
	}
}

type GetAppsOutput struct {
	Body struct {
		Apps []string `json:"apps"`
	}
}

func NewHandler(issuer string, conf *utils.Config) http.Handler {
	r := http.NewServeMux()

	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwks := goidc.JSONWebKeySet{
		Keys: []goidc.JSONWebKey{{
			KeyID:     "key_id",
			Key:       key,
			Algorithm: "RS256",
		}},
	}

	op, _ := provider.New(
		goidc.ProfileOpenID,
		issuer,
		func(_ context.Context) (goidc.JSONWebKeySet, error) {
			return jwks, nil
		},
		provider.WithStaticClient(&goidc.Client{}),
	)

	op.RegisterRoutes(r)
	registerApiRoutes(issuer, conf, r)

	r.Handle("/git", http.StripPrefix("/git", gitHandler(conf)))
	r.Handle("/webdav", &webdav.Handler{
		Prefix:     "/webdav",
		LockSystem: webdav.NewMemLS(),
		FileSystem: webdav.Dir(conf.String("dir")),
	})

	return r
}

func registerApiRoutes(issuer string, conf *utils.Config, r humago.Mux) {
	api := humago.New(r, huma.DefaultConfig("Smallweb API", "v1.0.0"))

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Tags:   []string{"Apps"},
		Path:   "/v1/apps",
		Servers: []*huma.Server{
			{URL: issuer},
		},
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
}

func gitHandler(conf *utils.Config) http.Handler {
	repoRoot := filepath.Join(conf.String("dir"))

	gitPath, err := exec.LookPath("git")
	if err != nil {
		panic("git binary not found in PATH")
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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/git-receive-pack") {
			http.Error(w, "pushes are disabled", http.StatusForbidden)
			return
		}

		fixChunked(r)

		gitHandler.ServeHTTP(w, r)
	})
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
