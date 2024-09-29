-- +goose Up
-- +goose StatementBegin
ALTER TABLE tokens ADD COLUMN admin BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE tokens SET admin = TRUE;
CREATE TABLE IF NOT EXISTS token_apps (
    token_id TEXT NOT NULL,
    app_name TEXT NOT NULL,
    PRIMARY KEY (token_id, app_name),
    FOREIGN KEY (token_id) REFERENCES tokens(id) ON DELETE CASCADE
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tokens DROP COLUMN all_apps_accesss;
DROP TABLE IF EXISTS token_apps;
-- +goose StatementEnd
