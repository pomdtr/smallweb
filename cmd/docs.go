package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "Generate smallweb cli documentation",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := buildDoc(cmd.Root())
			if err != nil {
				return fmt.Errorf("failed to generate docs: %w", err)
			}

			fmt.Println("# CLI Reference")
			fmt.Println()
			fmt.Println(doc)
			fmt.Println()
			fmt.Println("<!-- markdownlint-disable-file -->")

			return nil
		},
	}
	return cmd
}
func buildDoc(command *cobra.Command) (string, error) {
	var page strings.Builder
	err := doc.GenMarkdown(command, &page)
	if err != nil {
		return "", err
	}

	out := strings.Builder{}
	for _, line := range strings.Split(page.String(), "\n") {
		if strings.Contains(line, "SEE ALSO") {
			break
		}

		out.WriteString(line + "\n")
	}

	for _, child := range command.Commands() {
		if child.Name() == "help" {
			continue
		}

		childPage, err := buildDoc(child)
		if err != nil {
			return "", err
		}
		out.WriteString(childPage)
	}

	return out.String(), nil
}
