package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
)

type UserBranchRepository interface {
	// GetBranchIDs returns the branch IDs assigned to the user (non-owner path).
	GetBranchIDs(ctx context.Context, userID int64) ([]int64, error)
	// SetBranchIDs atomically replaces all branch assignments for the user.
	SetBranchIDs(ctx context.Context, userID int64, branchIDs []int64) error
	// HasAccess reports whether the user has an explicit branch assignment.
	HasAccess(ctx context.Context, userID int64, branchID int64) (bool, error)
}

type UserBranchRepo struct {
	db *pgxpool.Pool
}

func NewUserBranchRepo(db *pgxpool.Pool) *UserBranchRepo {
	return &UserBranchRepo{db: db}
}

func (r *UserBranchRepo) GetBranchIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.db.Query(ctx,
		`SELECT branch_id FROM user_branches WHERE user_id = $1 ORDER BY branch_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SetBranchIDs atomically deletes existing assignments then inserts the new set.
// A FK violation on branch_id is mapped to ErrInvalidInput (invalid branch).
func (r *UserBranchRepo) SetBranchIDs(ctx context.Context, userID int64, branchIDs []int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx,
		`DELETE FROM user_branches WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, branchID := range branchIDs {
		if _, err = tx.Exec(ctx,
			`INSERT INTO user_branches (user_id, branch_id) VALUES ($1, $2)`,
			userID, branchID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return apperrors.ErrInvalidInput
			}
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *UserBranchRepo) HasAccess(ctx context.Context, userID int64, branchID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_branches WHERE user_id = $1 AND branch_id = $2)`,
		userID, branchID).Scan(&exists)
	return exists, err
}
