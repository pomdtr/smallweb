package utils

import (
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
)

var RootDir string

func init() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current directory: %v", err)
	}

	for path.Dir(currentDir) != currentDir {
		if stat, err := os.Stat(filepath.Join(currentDir, ".smallweb")); err == nil && stat.IsDir() {
			RootDir = currentDir
			return
		}

		currentDir = path.Dir(currentDir)
	}

	if env, ok := os.LookupEnv("SMALLWEB_DIR"); ok {
		RootDir = env
		return
	}

	RootDir = filepath.Join(os.Getenv("HOME"), "smallweb")
}

func ConfigPath() string {
	return filepath.Join(RootDir, ".smallweb", "config.json")
}

func PluginDirs() []string {
	return []string{filepath.Join(RootDir, ".smallweb", "plugins"), filepath.Join(xdg.DataHome, "smallweb", "plugins")}
}

func DenoDir() string {
	return filepath.Join(xdg.CacheHome, "smallweb", "deno")
}
