package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/database"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func NewCmdToken(db *sql.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "token",
		Short:   "Manage api tokens",
		GroupID: CoreGroupID,
	}

	cmd.AddCommand(NewCmdTokenCreate(db))
	cmd.AddCommand(NewCmdTokenList(db))
	cmd.AddCommand(NewCmdTokenDelete(db))
	return cmd
}

func NewCmdTokenCreate(db *sql.DB) *cobra.Command {
	var flags struct {
		description string
		admin       bool
		app         []string
	}

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add", "new"},
		Short:   "Create a new token",
		Args:    cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(flags.app) == 0 && !flags.admin {
				return fmt.Errorf("either --admin or --app must be specified")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			value, public, secret, err := database.GenerateToken()
			if err != nil {
				return fmt.Errorf("failed to generate token: %v", err)
			}

			hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("failed to hash secret: %v", err)
			}

			token := database.Token{
				ID:          public,
				Description: flags.description,
				Hash:        hash,
				CreatedAt:   time.Now(),
				Admin:       flags.admin,
				Apps:        flags.app,
			}

			if err := database.InsertToken(db, token); err != nil {
				return fmt.Errorf("failed to insert token: %v", err)
			}

			if isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Println(value)
			} else {
				fmt.Print(value)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.description, "description", "d", "", "description of the token")
	cmd.MarkFlagRequired("description")
	cmd.Flags().BoolVar(&flags.admin, "admin", false, "admin token")
	cmd.Flags().StringSliceVarP(&flags.app, "app", "a", nil, "app token")
	cmd.RegisterFlagCompletionFunc("app", completeApp(utils.ExpandTilde("~/.smallweb/apps")))
	cmd.MarkFlagsMutuallyExclusive("admin", "app")

	return cmd
}

func NewCmdTokenList(db *sql.DB) *cobra.Command {
	var flags struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all tokens",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tokens, err := database.ListTokens(db)
			if err != nil {
				return fmt.Errorf("failed to list tokens: %v", err)
			}

			if flags.json {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetEscapeHTML(false)

				if isatty.IsTerminal(os.Stdout.Fd()) {
					encoder.SetIndent("", "  ")
				}

				if err := encoder.Encode(tokens); err != nil {
					return fmt.Errorf("failed to encode tokens: %w", err)
				}

				return nil
			}

			if len(tokens) == 0 {
				fmt.Println("No tokens found")
				return nil
			}

			var printer tableprinter.TablePrinter
			if isatty.IsTerminal(os.Stdout.Fd()) {
				width, _, err := term.GetSize(int(os.Stdout.Fd()))
				if err != nil {
					return fmt.Errorf("failed to get terminal size: %w", err)
				}

				printer = tableprinter.New(os.Stdout, true, width)
			} else {
				printer = tableprinter.New(os.Stdout, false, 0)
			}

			printer.AddHeader([]string{"ID", "Description", "Admin", "Apps", "Creation Time"})
			for _, token := range tokens {
				printer.AddField(token.ID)
				description := token.Description
				if description == "" {
					description = "N/A"
				}
				printer.AddField(description)
				printer.AddField(fmt.Sprintf("%t", token.Admin))
				if token.Admin {
					printer.AddField("<all>")
				} else {
					printer.AddField(strings.Join(token.Apps, ", "))
				}
				printer.AddField(token.CreatedAt.Format(time.RFC3339))
				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVarP(&flags.json, "json", "j", false, "output as JSON")
	return cmd
}

func NewCmdTokenDelete(db *sql.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   "Remove a token",
		Args:    cobra.ArbitraryArgs,
		Aliases: []string{"remove", "rm"},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			tokens, err := database.ListTokens(db)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, token := range tokens {
				completions = append(completions, fmt.Sprintf("%s\t%s", token.ID, token.Description))
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if err := database.DeleteToken(db, arg); err != nil {
					return fmt.Errorf("failed to delete token: %v", err)
				}

				cmd.Printf("Token %s removed\n", arg)
			}

			return nil
		},
	}

	return cmd
}
