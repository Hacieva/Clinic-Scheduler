-- +goose Up
-- +goose StatementBegin

-- Promote admin@clinic.local to owner role.
-- Idempotent: if already 'owner', UPDATE touches 0 rows.
UPDATE users
SET    role       = 'owner',
       updated_at = NOW()
WHERE  email = 'admin@clinic.local';

-- Assign owner to the default branch (branch id=1).
-- owner bypass logic skips branch filter regardless of this row,
-- but the entry is useful for explicit scoping if needed later.
-- ON CONFLICT DO NOTHING makes this idempotent.
INSERT INTO user_branches (user_id, branch_id)
SELECT id, 1
FROM   users
WHERE  email = 'admin@clinic.local'
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM user_branches
WHERE user_id = (SELECT id FROM users WHERE email = 'admin@clinic.local')
  AND branch_id = 1;

UPDATE users
SET    role       = 'admin',
       updated_at = NOW()
WHERE  email = 'admin@clinic.local';

-- +goose StatementEnd
