-- +goose Up
-- +goose StatementBegin
-- Make direction_id optional on services so catalog services don't require
-- a specialisation grouping to be created first.
ALTER TABLE services ALTER COLUMN direction_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Only safe to revert when no services have direction_id = NULL.
UPDATE services SET direction_id = 1 WHERE direction_id IS NULL;
ALTER TABLE services ALTER COLUMN direction_id SET NOT NULL;
-- +goose StatementEnd
