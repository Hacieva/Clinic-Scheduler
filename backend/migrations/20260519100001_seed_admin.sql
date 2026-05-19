-- +goose Up
-- +goose StatementBegin

-- Admin user: email=admin@clinic.local, password=changeme123 (bcrypt cost=12)
INSERT INTO users (email, password_hash, role, is_active)
VALUES (
    'admin@clinic.local',
    '$2a$12$aZxQek8QYMe1k/iLETUWHuUkKe2FL.gzBnbGbs.qwWrEQbYnQomVq',
    'admin',
    true
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM users WHERE email = 'admin@clinic.local';

-- +goose StatementEnd
