package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var fs embed.FS

func OpenDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(fs)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("failed to set dialect: %v", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	return db, nil
}
