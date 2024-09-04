package cmd

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/cli/go-gh/v2/pkg/tableprinter"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/smallweb/database"
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
	cmd.AddCommand(NewCmdTokenRemove(db))
	return cmd
}

func NewCmdTokenCreate(db *sql.DB) *cobra.Command {
	var flags struct {
		description string
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := generateToken(16)
			if err != nil {
				return fmt.Errorf("failed to generate token: %v", err)
			}

			hash, err := HashToken(value)
			if err != nil {
				return fmt.Errorf("failed to hash token: %v", err)
			}

			tokenID, err := gonanoid.New()
			if err != nil {
				return fmt.Errorf("failed to generate token ID: %v", err)
			}

			token := database.Token{
				ID:          tokenID,
				Description: flags.description,
				Hash:        hash,
				CreatedAt:   time.Now(),
			}

			dataHome := filepath.Join(xdg.DataHome, "smallweb")
			if err := os.MkdirAll(dataHome, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %v", err)
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
			dataHome := filepath.Join(xdg.DataHome, "smallweb")
			if err := os.MkdirAll(dataHome, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %v", err)
			}

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

			printer.AddHeader([]string{"ID", "Description", "Creation Time"})
			for _, token := range tokens {
				printer.AddField(token.ID)
				description := token.Description
				if description == "" {
					description = "N/A"
				}
				printer.AddField(description)
				printer.AddField(token.CreatedAt.Format(time.RFC3339))
				printer.EndRow()
			}

			return printer.Render()
		},
	}

	cmd.Flags().BoolVarP(&flags.json, "json", "j", false, "output as JSON")
	return cmd
}

func NewCmdTokenRemove(db *sql.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := database.DeleteToken(db, args[0]); err != nil {
				return fmt.Errorf("failed to delete token: %v", err)
			}

			cmd.Println("Token removed")
			return nil
		},
	}

	return cmd
}

func generateToken(n int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i := 0; i < n; i++ {
		bytes[i] = letters[bytes[i]%byte(len(letters))]
	}

	return string(bytes), nil
}

func HashToken(token string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(token), 14)
	return string(bytes), err
}
