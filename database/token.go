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
	Admin       bool      `json:"admin"`
	Apps        []string  `json:"apps"`
}

type TokenApp struct {
	TokenID string `json:"tokenID"`
	AppName string `json:"appName"`
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

func InsertToken(db *sql.DB, token Token) error {
	if token.Admin && len(token.Apps) > 0 {
		return fmt.Errorf("admin tokens cannot have apps")
	}

	if !token.Admin && len(token.Apps) == 0 {
		return fmt.Errorf("non-admin tokens must have apps")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO tokens (id, hash, description, createdAt, admin) VALUES (?, ?, ?, ?, ?)", token.ID, token.Hash, token.Description, token.CreatedAt, token.Admin)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, app := range token.Apps {
		_, err = tx.Exec("INSERT INTO token_apps (token_id, app_name) VALUES (?, ?)", token.ID, app)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func GetToken(db *sql.DB, id string) (Token, error) {
	token := Token{
		Apps: []string{},
	}
	err := db.QueryRow("SELECT id, hash, description, createdAt, admin FROM tokens WHERE id = ?", id).Scan(&token.ID, &token.Hash, &token.Description, &token.CreatedAt, &token.Admin)
	if err != nil {
		return token, err
	}

	rows, err := db.Query("SELECT app_name FROM token_apps WHERE token_id = ?", id)
	if err != nil {
		return token, err
	}
	defer rows.Close()

	for rows.Next() {
		var app string
		err := rows.Scan(&app)
		if err != nil {
			return token, err
		}

		token.Apps = append(token.Apps, app)
	}

	return token, err
}

func ListTokenApps(db *sql.DB) ([]TokenApp, error) {
	rows, err := db.Query("SELECT token_id, app_name FROM token_apps")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	apps := []TokenApp{}
	for rows.Next() {
		app := TokenApp{}
		err := rows.Scan(&app.TokenID, &app.AppName)
		if err != nil {
			return nil, err
		}

		apps = append(apps, app)
	}

	return apps, nil
}

func ListTokens(db *sql.DB) ([]Token, error) {
	tokenApps, err := ListTokenApps(db)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT id, hash, description, createdAt, admin FROM tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []Token{}
	for rows.Next() {
		token := Token{
			Apps: []string{},
		}
		err := rows.Scan(&token.ID, &token.Hash, &token.Description, &token.CreatedAt, &token.Admin)
		if err != nil {
			return nil, err
		}

		for _, app := range tokenApps {
			if app.TokenID == token.ID {
				token.Apps = append(token.Apps, app.AppName)
			}
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

func DeleteToken(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM tokens WHERE id = ?", id)
	return err
}
