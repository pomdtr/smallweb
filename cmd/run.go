package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	return &cobra.Command{
		Use:                "run <alias> [args...]",
		Short:              "Run a mod",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, err := NewWorker(args[0])
			if err != nil {
				return err
			}

			return worker.Run(args[1:])
		},
	}

}
