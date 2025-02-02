package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func NewCmdConfig() *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:   "config <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := k.Get(args[0])
			if v == nil {
				return fmt.Errorf("key %q not found", args[0])
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)
				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(v); err != nil {
					return err
				}

				return nil
			}

			fmt.Println(v)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.json, "json", false, "Output as JSON")

	return cmd
}
