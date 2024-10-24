package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pomdtr/smallweb/utils"
	"golang.org/x/crypto/bcrypt"
)

type Token struct {
	ID          string    `json:"id"`
	Hash        []byte    `json:"hash"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	App         string    `json:"app"`
}

func tokenDir() string {
	return filepath.Join(utils.DataDir(), "tokens")
}

func tokenPath(publicID string) string {
	return filepath.Join(tokenDir(), fmt.Sprintf("%s.json", publicID))
}

// Lengths for the public and secret parts
const publicPartLength = 16 // 16 characters for public part
const secretPartLength = 59 // 43 characters for secret part

const TokenPrefix = "smallweb_pat"

func GenerateToken() (string, string, string, error) {
	// Generate public and secret parts with Base62 encoding
	publicPart, err := utils.GenerateBase62String(publicPartLength)
	if err != nil {
		return "", "", "", err
	}
	secretPart, err := utils.GenerateBase62String(secretPartLength)
	if err != nil {
		return "", "", "", err
	}

	// Assemble the token with the given prefix
	return fmt.Sprintf("%s_%s_%s", TokenPrefix, publicPart, secretPart), publicPart, secretPart, nil
}

func GetToken(publicID string) (Token, error) {
	tokenBytes, err := os.ReadFile(tokenPath(publicID))
	if err != nil {
		return Token{}, err
	}

	var token Token
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return Token{}, err
	}

	return token, nil
}

func CreateToken(token Token) error {
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return err
	}

	tokenPath := tokenPath(token.ID)
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		return err
	}

	if err := os.WriteFile(tokenPath, tokenBytes, 0600); err != nil {
		return err
	}

	return nil
}

func VerifyToken(token string, appname string) error {
	public, secret, err := ParseToken(token)
	if err != nil {
		return err
	}

	t, err := GetToken(public)
	if err != nil {
		return err
	}

	if t.App != appname {
		return fmt.Errorf("invalid token")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(t.Hash), []byte(secret)); err != nil {
		return err
	}

	return nil
}

func ParseToken(token string) (string, string, error) {
	if !strings.HasPrefix(token, TokenPrefix) {
		return "", "", fmt.Errorf("invalid token format")
	}

	parts := strings.Split(token, "_")
	if len(parts) != 4 {
		return "", "", fmt.Errorf("invalid token format")
	}

	public, secret := parts[2], parts[3]
	if len(public) != publicPartLength || len(secret) != secretPartLength {
		return "", "", fmt.Errorf("invalid token format")
	}

	return parts[2], parts[3], nil
}

func ListTokens() ([]Token, error) {
	entries, err := os.ReadDir(tokenDir())
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	tokens := make([]Token, 0)
	for _, entry := range entries {
		tokenBytes, err := os.ReadFile(filepath.Join(tokenDir(), entry.Name()))
		if err != nil {
			return nil, err
		}

		var token Token
		if err := json.Unmarshal(tokenBytes, &token); err != nil {
			return nil, err
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

func DeleteToken(id string) error {
	if err := os.Remove(tokenPath(id)); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}
