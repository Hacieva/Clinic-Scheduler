-- +goose Up
-- +goose StatementBegin

CREATE TABLE branches (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(200) NOT NULL,
    address    TEXT,
    phone      VARCHAR(30),
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO branches (name) VALUES ('Главный филиал');

ALTER TABLE doctors      ADD COLUMN branch_id BIGINT REFERENCES branches(id);
ALTER TABLE appointments ADD COLUMN branch_id BIGINT REFERENCES branches(id);

-- backfill: all existing doctors → default branch 1
UPDATE doctors SET branch_id = 1;

-- backfill: appointments inherit branch from their doctor
UPDATE appointments
SET branch_id = (
    SELECT branch_id FROM doctors WHERE doctors.id = appointments.doctor_id
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE appointments DROP COLUMN IF EXISTS branch_id;
ALTER TABLE doctors      DROP COLUMN IF EXISTS branch_id;
DROP TABLE IF EXISTS branches;

-- +goose StatementEnd
