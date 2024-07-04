package templates

import (
	"embed"
	"fmt"
	"path"

	"github.com/leaanthony/debme"
	"github.com/leaanthony/gosod"
)

//go:embed all:templates
var templateFS embed.FS

func List() ([]string, error) {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, err
	}
	var templates []string
	for _, entry := range entries {
		templates = append(templates, entry.Name())
	}

	return templates, nil
}

func Install(name string, dst string) error {
	root, err := debme.FS(templateFS, path.Join("templates", name))
	if err != nil {
		return fmt.Errorf("template %s not found", name)
	}

	return gosod.New(root).Extract(dst, nil)
}
