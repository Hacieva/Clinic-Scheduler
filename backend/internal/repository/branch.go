package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

type BranchRepository interface {
	List(ctx context.Context) ([]model.Branch, error)
	GetByID(ctx context.Context, id int64) (*model.Branch, error)
	Create(ctx context.Context, input CreateBranchInput) (*model.Branch, error)
	Update(ctx context.Context, id int64, input UpdateBranchInput) (*model.Branch, error)
	Deactivate(ctx context.Context, id int64) error
	HasActiveDoctors(ctx context.Context, id int64) (bool, error)
}

type CreateBranchInput struct {
	Name    string
	Address *string
	Phone   *string
}

type UpdateBranchInput = CreateBranchInput

type BranchRepo struct {
	db *pgxpool.Pool
}

func NewBranchRepo(db *pgxpool.Pool) *BranchRepo {
	return &BranchRepo{db: db}
}

func (r *BranchRepo) List(ctx context.Context) ([]model.Branch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, address, phone, is_active, created_at, updated_at
		FROM   branches
		ORDER  BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.Branch{}
	for rows.Next() {
		var b model.Branch
		if err := rows.Scan(&b.ID, &b.Name, &b.Address, &b.Phone,
			&b.IsActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

func (r *BranchRepo) GetByID(ctx context.Context, id int64) (*model.Branch, error) {
	b := &model.Branch{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, address, phone, is_active, created_at, updated_at
		FROM   branches WHERE id = $1`, id).
		Scan(&b.ID, &b.Name, &b.Address, &b.Phone, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BranchRepo) Create(ctx context.Context, input CreateBranchInput) (*model.Branch, error) {
	b := &model.Branch{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO branches (name, address, phone)
		VALUES ($1, $2, $3)
		RETURNING id, name, address, phone, is_active, created_at, updated_at`,
		input.Name, input.Address, input.Phone).
		Scan(&b.ID, &b.Name, &b.Address, &b.Phone, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *BranchRepo) Update(ctx context.Context, id int64, input UpdateBranchInput) (*model.Branch, error) {
	b := &model.Branch{}
	err := r.db.QueryRow(ctx, `
		UPDATE branches
		SET    name = $1, address = $2, phone = $3, updated_at = NOW()
		WHERE  id = $4
		RETURNING id, name, address, phone, is_active, created_at, updated_at`,
		input.Name, input.Address, input.Phone, id).
		Scan(&b.ID, &b.Name, &b.Address, &b.Phone, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BranchRepo) Deactivate(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE branches SET is_active = false, updated_at = NOW()
		WHERE  id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// HasActiveDoctors reports whether any active doctor is assigned to the given branch.
// Used by BranchService to guard deactivation.
func (r *BranchRepo) HasActiveDoctors(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM doctors WHERE branch_id = $1 AND is_active = true
		)`, id).Scan(&exists)
	return exists, err
}
