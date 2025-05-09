package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pomdtr/smallweb/internal/build"
	"github.com/spf13/cobra"
)

func NewCmdMcp() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mcp",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := server.NewMCPServer("smallweb", build.Version)

			tool := mcp.NewTool("hello_world",
				mcp.WithDescription("Say hello to someone"),
				mcp.WithString("name",
					mcp.Required(),
					mcp.Description("Name of the person to greet"),
				),
			)

			s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				name, ok := request.Params.Arguments["name"].(string)
				if !ok {
					return nil, errors.New("name must be a string")
				}

				return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
			})

			return server.ServeStdio(s)

		},
	}

	return cmd
}
