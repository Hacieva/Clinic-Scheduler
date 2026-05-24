-- +goose Up
-- +goose StatementBegin

-- Step 1: make doctor_id nullable — services now live in a global catalog
ALTER TABLE services ALTER COLUMN doctor_id DROP NOT NULL;

-- Step 2: free-text category for service grouping (e.g. "Diagnostics", "Consultations")
ALTER TABLE services ADD COLUMN IF NOT EXISTS category VARCHAR(255);

-- Step 3: junction — which doctor performs which catalog service
CREATE TABLE IF NOT EXISTS doctor_services (
    id         BIGSERIAL PRIMARY KEY,
    doctor_id  BIGINT NOT NULL REFERENCES doctors(id)  ON DELETE CASCADE,
    service_id BIGINT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(doctor_id, service_id)
);

-- Step 4: backfill junction from existing per-doctor services
INSERT INTO doctor_services (doctor_id, service_id)
SELECT doctor_id, id
FROM   services
WHERE  doctor_id IS NOT NULL
ON CONFLICT (doctor_id, service_id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS doctor_services;

ALTER TABLE services DROP COLUMN IF EXISTS category;

-- Restoring NOT NULL requires no NULLs; safe for rollback only when no catalog-only
-- services exist yet (i.e. rolled back before any catalog service was created).
UPDATE services SET doctor_id = NULL WHERE doctor_id IS NULL;
ALTER TABLE services ALTER COLUMN doctor_id SET NOT NULL;

-- +goose StatementEnd
