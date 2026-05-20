package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

type DirectionRepository interface {
	List(ctx context.Context) ([]model.Direction, error)
	GetByID(ctx context.Context, id int64) (*model.Direction, error)
	Create(ctx context.Context, name string, description *string) (*model.Direction, error)
	Update(ctx context.Context, id int64, name string, description *string) (*model.Direction, error)
	SoftDelete(ctx context.Context, id int64) error
}

type DirectionRepo struct {
	db *pgxpool.Pool
}

func NewDirectionRepo(db *pgxpool.Pool) *DirectionRepo {
	return &DirectionRepo{db: db}
}

func (r *DirectionRepo) List(ctx context.Context) ([]model.Direction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, description, is_active, created_at, updated_at
		FROM   directions
		ORDER  BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Direction
	for rows.Next() {
		var d model.Direction
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []model.Direction{}
	}
	return result, nil
}

func (r *DirectionRepo) GetByID(ctx context.Context, id int64) (*model.Direction, error) {
	d := &model.Direction{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, is_active, created_at, updated_at
		FROM   directions
		WHERE  id = $1`, id).
		Scan(&d.ID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

func (r *DirectionRepo) Create(ctx context.Context, name string, description *string) (*model.Direction, error) {
	d := &model.Direction{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO directions (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, is_active, created_at, updated_at`,
		name, description).
		Scan(&d.ID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DirectionRepo) Update(ctx context.Context, id int64, name string, description *string) (*model.Direction, error) {
	d := &model.Direction{}
	err := r.db.QueryRow(ctx, `
		UPDATE directions
		SET    name = $1, description = $2, updated_at = NOW()
		WHERE  id = $3
		RETURNING id, name, description, is_active, created_at, updated_at`,
		name, description, id).
		Scan(&d.ID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

func (r *DirectionRepo) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE directions SET is_active = false, updated_at = NOW()
		WHERE  id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}
