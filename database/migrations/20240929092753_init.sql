-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tokens (
		id TEXT PRIMARY KEY,
		hash TEXT NOT NULL,
		description TEXT,
		createdAt TIMESTAMP NOT NULL
	);

CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		domain TEXT NOT NULL,
		createdAt TIMESTAMP NOT NULL,
		expiresAt TIMESTAMP NOT NULL
	);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd
