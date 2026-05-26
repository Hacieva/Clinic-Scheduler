-- +goose Up
-- +goose StatementBegin
-- direction_id was originally NOT NULL, but services now live in a global catalog
-- and may have no direction_id. Appointments must match.
ALTER TABLE appointments ALTER COLUMN direction_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Only safe when no appointments have direction_id = NULL.
UPDATE appointments SET direction_id = 1 WHERE direction_id IS NULL;
ALTER TABLE appointments ALTER COLUMN direction_id SET NOT NULL;
-- +goose StatementEnd
