package database

import (
	"database/sql"
	"fmt"

	"github.com/pomdtr/smallweb/utils"
	_ "modernc.org/sqlite"
)

func OpenDB(dbPath string) (*sql.DB, error) {
	if !utils.FileExists(dbPath) {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %v", err)
		}

		if err := CreateTokenTable(db); err != nil {
			return nil, fmt.Errorf("failed to create token table: %v", err)
		}

		if err := CreateSessionTable(db); err != nil {
			return nil, fmt.Errorf("failed to create session table: %v", err)
		}

		return db, nil
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	return db, nil
}
