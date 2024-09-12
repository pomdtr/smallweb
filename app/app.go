package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pomdtr/smallweb/utils"
	"github.com/tailscale/hujson"
)

type CronJob struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Schedule    string   `json:"schedule"`
	Args        []string `json:"args"`
}

type AppConfig struct {
	Entrypoint    string    `json:"entrypoint"`
	Root          string    `json:"root"`
	Private       bool      `json:"private"`
	PublicRoutes  []string  `json:"publicRoutes"`
	PrivateRoutes []string  `json:"privateRoutes"`
	Crons         []CronJob `json:"crons"`
}

type App struct {
	Dir    string `json:"-"`
	Env    map[string]string
	Config AppConfig
}

func (me App) Name() string {
	return filepath.Base(me.Dir)
}

func (me *App) Root() string {
	if me.Config.Root != "" {
		return filepath.Join(me.Dir, me.Config.Root)
	} else {
		return me.Dir
	}
}

func ListApps(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("could not read directory: %v", err)
	}

	apps := make([]string, 0)
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

func LoadApp(dir string) (App, error) {
	app := App{
		Dir: dir,
		Env: make(map[string]string),
	}

	if dotenvPath := filepath.Join(dir, ".env"); utils.FileExists(dotenvPath) {
		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			app.Env[key] = value
		}
	}

	if configPath := filepath.Join(dir, "smallweb.json"); utils.FileExists(configPath) {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return app, nil
	}

	if configPath := filepath.Join(dir, "smallweb.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read deno.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize deno.jsonc: %v", err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return app, nil
	}

	if configPath := filepath.Join(dir, "deno.json"); utils.FileExists(configPath) {
		denoConfigBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read deno.json: %v", err)
		}

		var denoConfig map[string]json.RawMessage
		if err := json.Unmarshal(denoConfigBytes, &denoConfig); err != nil {
			return App{}, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		configBytes, ok := denoConfig["smallweb"]
		if !ok {
			return app, nil
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return app, nil
	}

	if configPath := filepath.Join(dir, "deno.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read deno.json: %v", err)
		}

		denoConfigBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize deno.jsonc: %v", err)
		}

		var denoConfig map[string]json.RawMessage
		if err := json.Unmarshal(denoConfigBytes, &denoConfig); err != nil {
			return App{}, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		configBytes, ok := denoConfig["smallweb"]
		if !ok {
			return app, nil
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal deno.json: %v", err)
		}

		return app, nil
	}

	return app, nil
}

func (me App) Entrypoint() string {
	if strings.HasPrefix(me.Config.Entrypoint, "smallweb:") {
		return me.Config.Entrypoint
	}

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
			return path
		}
	}

	return "smallweb:static"
}
