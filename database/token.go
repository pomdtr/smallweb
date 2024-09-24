package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pomdtr/smallweb/utils"
)

type Token struct {
	ID          string    `json:"id"`
	Hash        []byte    `json:"hash"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
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

func CreateTokenTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS tokens (
		id TEXT PRIMARY KEY,
		hash TEXT NOT NULL,
		description TEXT,
		createdAt TIMESTAMP NOT NULL
	)`)

	return err
}

func InsertToken(db *sql.DB, token Token) error {
	_, err := db.Exec("INSERT INTO tokens (id, hash, description, createdAt) VALUES (?, ?, ?, ?)", token.ID, token.Hash, token.Description, token.CreatedAt)
	return err
}

func GetToken(db *sql.DB, id string) (Token, error) {
	token := Token{}
	err := db.QueryRow("SELECT id, hash, description, createdAt FROM tokens WHERE id = ?", id).Scan(&token.ID, &token.Hash, &token.Description, &token.CreatedAt)
	return token, err
}

func ListTokens(db *sql.DB) ([]Token, error) {
	rows, err := db.Query("SELECT id, hash, description, createdAt FROM tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []Token{}
	for rows.Next() {
		token := Token{}
		err := rows.Scan(&token.ID, &token.Hash, &token.Description, &token.CreatedAt)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func DeleteToken(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM tokens WHERE id = ?", id)
	return err
}
