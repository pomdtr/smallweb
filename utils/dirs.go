package utils

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

func RootDir() string {
	if env, ok := os.LookupEnv("SMALLWEB_DIR"); ok {
		return env
	}

	return filepath.Join(os.Getenv("HOME"), "smallweb")
}

func ConfigPath() string {
	return filepath.Join(RootDir(), ".smallweb", "config.json")
}

func PluginDirs() []string {
	return []string{filepath.Join(RootDir(), ".smallweb", "plugins"), filepath.Join(xdg.DataHome, "smallweb", "plugins")}
}

func DataDir() string {
	return filepath.Join(RootDir(), ".smallweb", "data")
}
