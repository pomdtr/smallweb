package utils

import (
	"os"
	"path/filepath"
)

func FindConfigPath(rootDir string) string {
	for _, candidate := range []string{"config.json", "config.jsonc"} {
		configPath := filepath.Join(rootDir, ".smallweb", candidate)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return filepath.Join(rootDir, ".smallweb/config.json")
}
