package database

import (
	"database/sql"
	"time"
)

type Token struct {
	ID          string    `json:"id"`
	Hash        []byte    `json:"hash"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
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
