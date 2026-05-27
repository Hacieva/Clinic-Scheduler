-- +goose Up
-- +goose StatementBegin

-- branches.name must be unique so ON CONFLICT (name) works in the importer.
ALTER TABLE branches
    ADD CONSTRAINT branches_name_unique UNIQUE (name);

-- doctors.import_source_id stores the source spreadsheet key (D001, V003…)
-- so the importer can upsert deterministically on re-runs.
ALTER TABLE doctors
    ADD COLUMN import_source_id VARCHAR(20) NULL;
ALTER TABLE doctors
    ADD CONSTRAINT doctors_import_source_id_unique UNIQUE (import_source_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE doctors    DROP CONSTRAINT IF EXISTS doctors_import_source_id_unique;
ALTER TABLE doctors    DROP COLUMN IF EXISTS import_source_id;
ALTER TABLE branches   DROP CONSTRAINT IF EXISTS branches_name_unique;

-- +goose StatementEnd
