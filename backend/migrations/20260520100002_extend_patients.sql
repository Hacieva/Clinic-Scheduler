-- +goose Up
-- +goose StatementBegin

-- Nullable fields — existing rows get NULL, existing INSERTs unaffected.
ALTER TABLE patients ADD COLUMN date_of_birth DATE;
ALTER TABLE patients ADD COLUMN email         VARCHAR(255);
ALTER TABLE patients ADD COLUMN comment       TEXT;

-- NOT NULL with DEFAULT: existing rows backfilled with 'telegram_bot' (PostgreSQL 11+
-- stores the default in catalog without a table rewrite, so this is safe on large tables).
-- Existing bot/admin INSERTs that omit 'source' will receive the default automatically.
ALTER TABLE patients ADD COLUMN source VARCHAR(50) NOT NULL DEFAULT 'telegram_bot';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE patients DROP COLUMN IF EXISTS source;
ALTER TABLE patients DROP COLUMN IF EXISTS comment;
ALTER TABLE patients DROP COLUMN IF EXISTS email;
ALTER TABLE patients DROP COLUMN IF EXISTS date_of_birth;

-- +goose StatementEnd
