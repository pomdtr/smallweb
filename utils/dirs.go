package utils

import (
	"os"
	"path/filepath"
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

func DataDir() string {
	return filepath.Join(RootDir(), ".smallweb", "data")
}
