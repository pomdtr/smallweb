package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pomdtr/smallweb/config"
	"github.com/pomdtr/smallweb/utils"
	"github.com/tailscale/hujson"
)

type AppConfig struct {
	Entrypoint string    `json:"entrypoint,omitempty"`
	Root       string    `json:"root,omitempty"`
	Admin      bool      `json:"admin,omitempty"`
	Crons      []CronJob `json:"crons,omitempty"`
}

type CronJob struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Schedule    string   `json:"schedule"`
	Args        []string `json:"args"`
}

type App struct {
	Name   string            `json:"name"`
	Dir    string            `json:"dir,omitempty"`
	URL    string            `json:"url"`
	Env    map[string]string `json:"-"`
	Config AppConfig         `json:"-"`
}

func (me *App) Root() string {
	if me.Config.Root != "" {
		return filepath.Join(me.Dir, me.Config.Root)
	} else {
		return me.Dir
	}
}

var APP_REGEX = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

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

		if !APP_REGEX.MatchString(entry.Name()) {
			continue
		}

		apps = append(apps, entry.Name())
	}

	return apps, nil
}

func LoadApp(name string, conf config.Config) (App, error) {
	if !APP_REGEX.MatchString(name) {
		return App{}, fmt.Errorf("invalid app name: %s", name)
	}

	appdir := filepath.Join(conf.Dir, name)
	if !utils.FileExists(appdir) {
		return App{}, fmt.Errorf("directory does not exist: %s", appdir)
	}

	app := App{
		Name: name,
		Dir:  appdir,
		URL:  fmt.Sprintf("https://%s.%s/", name, conf.Domain),
		Env:  make(map[string]string),
	}

	if dotenvPath := filepath.Join(conf.Dir, ".env"); utils.FileExists(dotenvPath) {
		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			app.Env[key] = value
		}
	}

	if dotenvPath := filepath.Join(appdir, ".env"); utils.FileExists(dotenvPath) {
		dotenv, err := godotenv.Read(dotenvPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read .env: %v", err)
		}

		for key, value := range dotenv {
			app.Env[key] = value
		}
	}

	if configPath := filepath.Join(appdir, "smallweb.json"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize smallweb.jsonc: %v", err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal smallweb.json: %v", err)
		}

		return app, nil
	}

	if configPath := filepath.Join(appdir, "smallweb.jsonc"); utils.FileExists(configPath) {
		rawBytes, err := os.ReadFile(configPath)
		if err != nil {
			return App{}, fmt.Errorf("could not read smallweb.json: %v", err)
		}

		configBytes, err := hujson.Standardize(rawBytes)
		if err != nil {
			return App{}, fmt.Errorf("could not standardize smallweb.jsonc: %v", err)
		}

		if err := json.Unmarshal(configBytes, &app.Config); err != nil {
			return App{}, fmt.Errorf("could not unmarshal smallweb.json: %v", err)
		}

		return app, nil
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
			return path
		}
	}

	return "jsr:@smallweb/file-server@0.4.1"
}
