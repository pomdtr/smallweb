package cmd

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pomdtr/smallweb/client"
	"github.com/spf13/cobra"
)

//go:embed examples/*
var examples embed.FS

func NewCmdInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize your smallweb directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// copy everything from the embed fs to the SMALLWEB_ROOT
			fs.WalkDir(examples, "examples", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				dst := filepath.Join(client.SMALLWEB_ROOT, strings.TrimPrefix(path, "examples"))
				if d.IsDir() {
					return os.MkdirAll(dst, 0755)
				}

				content, err := examples.ReadFile(path)
				if err != nil {
					return err
				}

				return os.WriteFile(dst, content, 0644)
			})

			return nil
		},
	}
}
