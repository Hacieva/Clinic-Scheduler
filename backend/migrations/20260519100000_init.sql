-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Directions (медицинские направления)
CREATE TABLE directions (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users (только admin и doctor)
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL CHECK (role IN ('admin', 'doctor')),
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_email_lowercase CHECK (email = lower(email))
);

-- Doctors
CREATE TABLE doctors (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    first_name     VARCHAR(255) NOT NULL,
    last_name      VARCHAR(255) NOT NULL,
    middle_name    VARCHAR(255),
    cabinet        VARCHAR(50),
    branch_address TEXT,
    description    TEXT,
    photo_url      VARCHAR(500),
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Doctor ↔ Direction (M2M)
CREATE TABLE doctor_directions (
    id           BIGSERIAL PRIMARY KEY,
    doctor_id    BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    direction_id BIGINT NOT NULL REFERENCES directions(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(doctor_id, direction_id)
);

-- Services (услуга принадлежит конкретному врачу)
CREATE TABLE services (
    id               BIGSERIAL PRIMARY KEY,
    doctor_id        BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    direction_id     BIGINT NOT NULL REFERENCES directions(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
    price            DECIMAL(10, 2),
    is_active        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Doctor working hours (недельный шаблон)
CREATE TABLE doctor_working_hours (
    id          BIGSERIAL PRIMARY KEY,
    doctor_id   BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    day_of_week INT  NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time  TIME NOT NULL,
    end_time    TIME NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (start_time < end_time),
    UNIQUE(doctor_id, day_of_week, start_time, end_time)
);

-- Doctor schedule exceptions (исключения имеют приоритет над шаблоном)
CREATE TABLE doctor_schedule_exceptions (
    id          BIGSERIAL PRIMARY KEY,
    doctor_id   BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    date        DATE NOT NULL,
    type        VARCHAR(50) NOT NULL CHECK (type IN ('day_off', 'custom_working_hours')),
    start_time  TIME,
    end_time    TIME,
    comment     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(doctor_id, date)
);

-- Patients (нет аккаунта, идентифицируется по telegram_user_id и phone)
CREATE TABLE patients (
    id                 BIGSERIAL PRIMARY KEY,
    telegram_user_id   BIGINT UNIQUE,
    telegram_username  VARCHAR(255),
    full_name          VARCHAR(255) NOT NULL,
    phone              VARCHAR(20)  NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT patients_phone_format CHECK (phone ~ '^\+?[0-9\s\-\(\)]{7,20}$')
);

-- Appointments
CREATE TABLE appointments (
    id              BIGSERIAL PRIMARY KEY,
    patient_id      BIGINT NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    doctor_id       BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    service_id      BIGINT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    direction_id    BIGINT NOT NULL REFERENCES directions(id) ON DELETE CASCADE,
    start_at        TIMESTAMPTZ NOT NULL,
    end_at          TIMESTAMPTZ NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'created' CHECK (status IN (
                        'created', 'confirmed', 'cancelled_by_patient',
                        'cancelled_by_admin', 'completed', 'no_show'
                    )),
    source          VARCHAR(50) NOT NULL CHECK (source IN ('telegram_bot', 'admin_panel')),
    patient_comment TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    EXCLUDE USING GIST (
        doctor_id WITH =,
        tstzrange(start_at, end_at) WITH &&
    ) WHERE (status IN ('created', 'confirmed'))
);

-- Appointment status history
CREATE TABLE appointment_status_history (
    id                  BIGSERIAL PRIMARY KEY,
    appointment_id      BIGINT NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
    old_status          VARCHAR(50),
    new_status          VARCHAR(50) NOT NULL,
    changed_by_user_id  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    changed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    comment             TEXT
);

-- Audit logs
CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action      VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50)  NOT NULL,
    entity_id   BIGINT,
    old_values  JSONB,
    new_values  JSONB,
    ip_address  VARCHAR(50),
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Bot FSM sessions (хранятся в БД, не в памяти)
CREATE TABLE bot_sessions (
    id               BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL UNIQUE,
    state            VARCHAR(100) NOT NULL,
    data             JSONB NOT NULL DEFAULT '{}',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_doctors_is_active        ON doctors(is_active) WHERE is_active = true;
CREATE INDEX idx_doctors_user_id          ON doctors(user_id);
CREATE INDEX idx_services_doctor_id       ON services(doctor_id);
CREATE INDEX idx_services_direction_id    ON services(direction_id);
CREATE INDEX idx_dwh_doctor_id            ON doctor_working_hours(doctor_id);
CREATE INDEX idx_dse_doctor_id            ON doctor_schedule_exceptions(doctor_id);
CREATE INDEX idx_appointments_doctor_start ON appointments(doctor_id, start_at);
CREATE INDEX idx_appointments_patient_id  ON appointments(patient_id);
CREATE INDEX idx_appointments_status      ON appointments(status);
CREATE INDEX idx_patients_telegram        ON patients(telegram_user_id);
CREATE INDEX idx_audit_logs_user_created  ON audit_logs(user_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS bot_sessions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS appointment_status_history;
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS doctor_schedule_exceptions;
DROP TABLE IF EXISTS doctor_working_hours;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS doctor_directions;
DROP TABLE IF EXISTS doctors;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS directions;
DROP EXTENSION IF EXISTS btree_gist;

-- +goose StatementEnd
