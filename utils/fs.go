package utils

import (
	"os"
	"strings"
)

func FileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func ExpandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		return strings.Replace(p, "~", os.Getenv("HOME"), 1)
	}
	return p
}

func AddTilde(p string) string {
	home := os.Getenv("HOME")
	if strings.HasPrefix(p, home) {
		return strings.Replace(p, home, "~", 1)
	}
	return p
}
