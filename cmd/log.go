package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/pomdtr/smallweb/client"
	"github.com/spf13/cobra"
)

func NewCmdLog() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "log <app>",
		Aliases: []string{"logs"},
		Short:   "View logs",
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			apps, err := listApps()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			return apps, cobra.ShellCompDirectiveNoFileComp

		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app := args[0]
			logFile := path.Join(client.SmallwebDir, "logs", app+".log")
			f, err := os.Open(logFile)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("log file not found")
				}

				return fmt.Errorf("failed to open log file: %v", err)
			}
			defer f.Close()

			follow, _ := cmd.Flags().GetBool("follow")
			if follow {
				f.Seek(0, io.SeekEnd)
				reader := bufio.NewReader(f)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							time.Sleep(100 * time.Millisecond)
							continue
						}

						return fmt.Errorf("failed to read log file: %v", err)
					}

					fmt.Print(line)
				}
			}

			_, err = io.Copy(os.Stdout, f)
			if err != nil {
				return fmt.Errorf("failed to copy log file to stdout: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("follow", "f", false, "Follow the log file")
	return cmd
}
