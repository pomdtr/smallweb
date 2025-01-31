package utils

import (
	"os"
	"strings"
)

func FileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func AddTilde(p string) string {
	home := os.Getenv("HOME")
	if strings.HasPrefix(p, home) {
		return strings.Replace(p, home, "~", 1)
	}
	return p
}
