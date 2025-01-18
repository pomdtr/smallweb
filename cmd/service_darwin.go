//go:build darwin

package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"text/template"
)

//go:embed embed/smallweb.plist
var serviceConfigBytes []byte
var serviceConfig = template.Must(template.New("service").Parse(string(serviceConfigBytes)))

func GetService(args []string) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}

	writer := &strings.Builder{}
	if err := serviceConfig.Execute(writer, map[string]any{
		"SmallwebDir": k.String("dir"),
		"ExecPath":    execPath,
		"Args":        args,
		"HomeDir":     os.Getenv("HOME"),
	}); err != nil {
		return "", fmt.Errorf("failed to write service file: %v", err)
	}

	return writer.String(), nil
}
