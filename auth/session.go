package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/pomdtr/smallweb/utils"
)

type Session struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func sessionDir() string {
	return filepath.Join(utils.DataDir(), "sessions")
}

func sessionPath(sessionID string) string {
	return filepath.Join(sessionDir(), fmt.Sprintf("%s.json", sessionID))
}

func CreateSession(email string, domain string) (string, error) {
	sessionID, err := gonanoid.New()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := Session{
		ID:        sessionID,
		Email:     email,
		Domain:    domain,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	}

	if err := SaveSession(session); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}

	if err := DeleteExpiredSessions(); err != nil {
		return "", fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return sessionID, nil
}

func SaveSession(session Session) error {
	sessionPath := sessionPath(session.ID)
	if err := os.MkdirAll(sessionDir(), 0700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	sessionBytes, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionPath, sessionBytes, 0600); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	return nil
}

func DeleteSession(sessionID string) error {
	sessionPath := sessionPath(sessionID)
	if err := os.Remove(sessionPath); err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}

	return nil
}

func DeleteExpiredSessions() error {
	entries, err := os.ReadDir(sessionDir())
	if err != nil {
		return fmt.Errorf("failed to read sessions: %w", err)
	}

	for _, entry := range entries {
		sessionPath := filepath.Join(sessionDir(), entry.Name())
		sessionBytes, err := os.ReadFile(sessionPath)
		if err != nil {
			return fmt.Errorf("failed to read session: %w", err)
		}

		var session Session
		if err := json.Unmarshal(sessionBytes, &session); err != nil {
			return fmt.Errorf("failed to unmarshal session: %w", err)
		}

		if session.ExpiresAt.Before(time.Now()) {
			if err := DeleteSession(session.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func GetSession(sessionID string) (Session, error) {
	sessionPath := sessionPath(sessionID)
	sessionBytes, err := os.ReadFile(sessionPath)
	if err != nil {
		return Session{}, fmt.Errorf("failed to read session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(sessionBytes, &session); err != nil {
		return Session{}, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return session, nil
}
