-- +goose Up
-- +goose StatementBegin

-- Doctor employment/presence relationship
CREATE TYPE doctor_kind AS ENUM (
    'staff',    -- permanent or regular staff
    'visiting'  -- comes on specific scheduled dates
);

-- How patients access a doctor
CREATE TYPE booking_mode AS ENUM (
    'appointment_only', -- slot-based booking only
    'queue_only',       -- walk-in queue only
    'mixed'             -- both simultaneously
);

-- Patient age group for services and assignments
CREATE TYPE patient_audience AS ENUM (
    'adult',
    'child',
    'both'
);

-- doctors: add workflow type, booking mode, and optional audience hint
ALTER TABLE doctors
    ADD COLUMN doctor_kind   doctor_kind     NOT NULL DEFAULT 'staff',
    ADD COLUMN booking_mode  booking_mode    NOT NULL DEFAULT 'appointment_only',
    ADD COLUMN audience      patient_audience NULL;    -- NULL = no doctor-level restriction

-- services: add optional Medlock service code for deduplication and future sync
ALTER TABLE services
    ADD COLUMN code VARCHAR(20) NULL;
ALTER TABLE services
    ADD CONSTRAINT services_code_unique UNIQUE (code);

-- doctor_services: add patient_type per assignment (authoritative audience source)
ALTER TABLE doctor_services
    ADD COLUMN patient_type patient_audience NOT NULL DEFAULT 'both';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE doctor_services    DROP COLUMN IF EXISTS patient_type;
ALTER TABLE services           DROP CONSTRAINT IF EXISTS services_code_unique;
ALTER TABLE services           DROP COLUMN IF EXISTS code;
ALTER TABLE doctors            DROP COLUMN IF EXISTS audience;
ALTER TABLE doctors            DROP COLUMN IF EXISTS booking_mode;
ALTER TABLE doctors            DROP COLUMN IF EXISTS doctor_kind;

DROP TYPE IF EXISTS patient_audience;
DROP TYPE IF EXISTS booking_mode;
DROP TYPE IF EXISTS doctor_kind;

-- +goose StatementEnd
