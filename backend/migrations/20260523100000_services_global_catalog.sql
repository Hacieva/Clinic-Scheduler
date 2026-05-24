-- +goose Up
-- +goose StatementBegin

ALTER TABLE doctors ADD COLUMN IF NOT EXISTS phone VARCHAR(20);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE doctors DROP COLUMN IF EXISTS phone;

-- +goose StatementEnd
