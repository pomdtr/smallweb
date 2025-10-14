package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	for _, candidate := range []string{"config.json", "config.jsonc"} {
		configPath := filepath.Join(configDir, "smallweb", candidate)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	return filepath.Join(configDir, ".config", "smallweb", "config.jsonc"), nil
}
