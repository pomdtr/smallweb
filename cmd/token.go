package cmd

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

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

func NewCmdToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Generate a random token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := generateToken(16)
			if err != nil {
				return fmt.Errorf("failed to generate token: %w", err)
			}

			if isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Println(token)
			} else {
				fmt.Print(token)
			}

			return nil
		},
	}

	return cmd
}
