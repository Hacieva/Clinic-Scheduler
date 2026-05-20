package session

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store backed by the bot_sessions table.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Get(ctx context.Context, telegramUserID int64) (*Data, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx,
		`SELECT data FROM bot_sessions WHERE telegram_user_id = $1`,
		telegramUserID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Replace is an atomic UPSERT: inserts if the user has no session, updates otherwise.
// Updates both the top-level state column and the full data JSONB.
func (s *PostgresStore) Replace(ctx context.Context, telegramUserID int64, data *Data) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO bot_sessions (telegram_user_id, state, data, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (telegram_user_id) DO UPDATE
		 SET state = EXCLUDED.state,
		     data  = EXCLUDED.data,
		     updated_at = NOW()`,
		telegramUserID, data.State, raw,
	)
	return err
}

// Delete removes the session row. Returns nil if the row did not exist.
func (s *PostgresStore) Delete(ctx context.Context, telegramUserID int64) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM bot_sessions WHERE telegram_user_id = $1`,
		telegramUserID,
	)
	return err
}
