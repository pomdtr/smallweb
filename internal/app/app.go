package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsops/sops/v3/decrypt"
	"github.com/joho/godotenv"
	"github.com/pomdtr/smallweb/internal/utils"
)

var (
	ErrAppNotFound = errors.New("app not found")
)

type Config struct {
	Entrypoint        string            `json:"entrypoint,omitempty" mapstructure:"entrypoint"`
	Root              string            `json:"root,omitempty" mapstructure:"root"`
	Crons             []CronJob         `json:"crons,omitempty" mapstructure:"crons"`
	AdditionalDomains []string          `json:"additionalDomains" mapstructure:"additionalDomains"`
	AuthorizedKeys    []string          `json:"authorizedKeys" mapstructure:"authorizedKeys"`
	Env               map[string]string `json:"env,omitempty" mapstructure:"env"`
}

type CronJob struct {
	Description string   `json:"description" mapstructure:"description"`
	Schedule    string   `json:"schedule" mapstructure:"schedule"`
	Args        []string `json:"args" mapstructure:"args"`
}

type App struct {
	Name   string
	Dir    string
	Config Config
	dotenv map[string]string
}

func (me *App) Root() string {
	root := me.Dir
	if me.Config.Root != "" {
		return filepath.Join(root, me.Config.Root)
	}

	if me.Config.Entrypoint != "" {
		return root
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		if utils.FileExists(filepath.Join(root, candidate)) {
			return root
		}
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		if utils.FileExists(filepath.Join(root, "dist", candidate)) {
			return filepath.Join(root, "dist")
		}
	}

	if utils.FileExists(filepath.Join(root, "dist", "index.html")) {
		return filepath.Join(root, "dist")
	}

	return root
}

func (me *App) DataDir() string {
	dir := filepath.Join(me.Root(), "data")
	if fi, err := os.Lstat(dir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if root, err := os.Readlink(dir); err == nil {
			dir = filepath.Join(filepath.Dir(dir), root)
		}
	}

	return dir
}

func LoadApp(appDir string, config Config) (App, error) {
	app := App{
		Name:   filepath.Base(appDir),
		Dir:    appDir,
		dotenv: make(map[string]string),
		Config: config,
	}

	if dotenvPath := filepath.Join(appDir, ".env"); utils.FileExists(dotenvPath) {
		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			app.dotenv[key] = value
		}
	}

	for _, secretPath := range []string{
		filepath.Join(appDir, "secrets.enc.env"),
		filepath.Join(appDir, "secrets.env"),
	} {
		if !utils.FileExists(secretPath) {
			continue
		}

		rawBytes, err := os.ReadFile(secretPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read file: %v", err)
		}

		dotenvBytes, err := decrypt.Data(rawBytes, "dotenv")
		if err != nil {
			return App{}, fmt.Errorf("could not decrypt %s: %v", secretPath, err)
		}

		dotenv, err := godotenv.Parse(bytes.NewReader(dotenvBytes))
		if err != nil {
			return App{}, fmt.Errorf("could not parse %s: %v", secretPath, err)
		}

		for key, value := range dotenv {
			app.dotenv[key] = value
		}

		break
	}

	return app, nil
}

func (me App) Entrypoint() string {
	if strings.HasPrefix(me.Config.Entrypoint, "jsr:") || strings.HasPrefix(me.Config.Entrypoint, "npm:") {
		return me.Config.Entrypoint
	}

	if strings.HasPrefix(me.Config.Entrypoint, "https://") || strings.HasPrefix(me.Config.Entrypoint, "http://") {
		return me.Config.Entrypoint
	}

	if me.Config.Entrypoint != "" {
		return filepath.Join(me.Root(), me.Config.Entrypoint)
	}

	for _, candidate := range []string{"main.js", "main.ts", "main.jsx", "main.tsx"} {
		path := filepath.Join(me.Root(), candidate)
		if utils.FileExists(path) {
			return fmt.Sprintf("file://%s", path)
		}
	}

	return "jsr:@smallweb/file-server@0.8.2"
}

func (me App) Env() []string {
	env := []string{}

	for k, v := range me.Config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range me.dotenv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	env = append(env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
	env = append(env, "DENO_NO_UPDATE_CHECK=1")

	// open telemetry
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "OTEL_") {
			env = append(env, value)
		}

		if strings.HasPrefix(value, "DENO_") {
			env = append(env, value)
		}
	}

	env = append(env, fmt.Sprintf("OTEL_SERVICE_NAME=%s", me.Name))

	return env
}

func List(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var apps []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if !entry.IsDir() {
			continue
		}

		apps = append(apps, entry.Name())
	}

	return apps, nil
}
