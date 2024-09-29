package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pomdtr/smallweb/api"
	"github.com/spf13/cobra"
)

func NewCmdOpenapi() *cobra.Command {
	var flags struct {
		types bool
	}

	cmd := &cobra.Command{
		Use:     "openapi",
		Short:   "Print the OpenAPI spec",
		GroupID: CoreGroupID,
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := api.GetSwagger()
			if err != nil {
				return fmt.Errorf("failed to get swagger: %w", err)
			}

			if flags.types {
				os.Stdout.WriteString("export default ")
				b, err := json.MarshalIndent(spec, "", "  ")
				if err != nil {
					return err
				}
				os.Stdout.Write(bytes.TrimRight(b, "\n"))
				os.Stdout.WriteString(" as const;\n")
				return nil
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			encoder.Encode(spec)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.types, "types", false, "Print typescript types")
	return cmd
}
