-- +goose Up
-- +goose StatementBegin

-- Expand role CHECK to include 'owner'.
-- Existing 'admin' and 'doctor' rows are unaffected — new constraint is a superset.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'doctor', 'owner'));

-- Junction table for assigning one or more branches to a user.
-- owner: no rows needed (sees all). admin: one or more rows.
-- Existing code does not reference this table — purely additive.
CREATE TABLE user_branches (
    user_id   BIGINT NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    branch_id BIGINT NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, branch_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS user_branches;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'doctor'));

-- +goose StatementEnd
