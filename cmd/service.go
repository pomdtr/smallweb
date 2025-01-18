package cmd

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmdService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Generate service file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := GetService()
			if err != nil {
				return fmt.Errorf("failed to get service file: %v", err)
			}

			fmt.Println(service)
			return nil
		},
	}

	return cmd
}
