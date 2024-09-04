package database

import (
	"database/sql"
	"time"
)

type Session struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func CreateSessionTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		domain TEXT NOT NULL,
		createdAt TIMESTAMP NOT NULL,
		expiresAt TIMESTAMP NOT NULL
	)`)

	return err
}

func InsertSession(db *sql.DB, session *Session) error {
	_, err := db.Exec("INSERT INTO sessions (id, email, domain, createdAt, expiresAt) VALUES (?, ?, ?, ?, ?)", session.ID, session.Email, session.Domain, session.CreatedAt, session.ExpiresAt)
	return err
}

func GetSession(db *sql.DB, id string) (*Session, error) {
	session := &Session{}
	err := db.QueryRow("SELECT id, email, domain, createdAt, expiresAt FROM sessions WHERE id = ?", id).Scan(&session.ID, &session.Email, &session.Domain, &session.CreatedAt, &session.ExpiresAt)
	return session, err
}

func UpdateSession(db *sql.DB, session *Session) error {
	_, err := db.Exec("UPDATE sessions SET email = ?, domain = ?, createdAt = ?, expiresAt = ? WHERE id = ?", session.Email, session.Domain, session.CreatedAt, session.ExpiresAt, session.ID)
	return err
}

func DeleteSession(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}
