package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/api"
	"github.com/spf13/cobra"
)

func NewCmdOpenapi() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "openapi",
		Short:  "Generate OpenAPI documentation",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := api.GetSwagger()
			if err != nil {
				return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
			}

			encoder := json.NewEncoder(os.Stdout)
			if isatty.IsTerminal(os.Stdout.Fd()) {
				encoder.SetIndent("", "  ")
			}

			return encoder.Encode(spec)
		},
	}
	return cmd
}
