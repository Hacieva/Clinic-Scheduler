package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

type ServiceRepository interface {
	ListByDoctor(ctx context.Context, doctorID int64) ([]model.Service, error)
	GetByID(ctx context.Context, id int64) (*model.Service, error)
	Create(ctx context.Context, input CreateServiceInput) (*model.Service, error)
	Update(ctx context.Context, id int64, input UpdateServiceInput) (*model.Service, error)
	SoftDelete(ctx context.Context, id int64) error
	GetDurationMinutes(ctx context.Context, serviceID int64) (int, error)
}

type CreateServiceInput struct {
	DoctorID        int64
	DirectionID     int64
	Name            string
	Description     *string
	DurationMinutes int
	Price           *int64
}

type UpdateServiceInput struct {
	DirectionID     int64
	Name            string
	Description     *string
	DurationMinutes int
	Price           *int64
}

type ServiceRepo struct {
	db *pgxpool.Pool
}

func NewServiceRepo(db *pgxpool.Pool) *ServiceRepo {
	return &ServiceRepo{db: db}
}

func (r *ServiceRepo) ListByDoctor(ctx context.Context, doctorID int64) ([]model.Service, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, doctor_id, direction_id, name, description,
		       duration_minutes, price, is_active, created_at, updated_at
		FROM   services
		WHERE  doctor_id = $1
		ORDER  BY id`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Service
	for rows.Next() {
		var s model.Service
		if err := rows.Scan(
			&s.ID, &s.DoctorID, &s.DirectionID, &s.Name, &s.Description,
			&s.DurationMinutes, &s.Price, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []model.Service{}
	}
	return result, nil
}

func (r *ServiceRepo) GetByID(ctx context.Context, id int64) (*model.Service, error) {
	var s model.Service
	err := r.db.QueryRow(ctx, `
		SELECT id, doctor_id, direction_id, name, description,
		       duration_minutes, price, is_active, created_at, updated_at
		FROM   services
		WHERE  id = $1`, id).
		Scan(&s.ID, &s.DoctorID, &s.DirectionID, &s.Name, &s.Description,
			&s.DurationMinutes, &s.Price, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepo) Create(ctx context.Context, input CreateServiceInput) (*model.Service, error) {
	var s model.Service
	err := r.db.QueryRow(ctx, `
		INSERT INTO services (doctor_id, direction_id, name, description, duration_minutes, price)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, doctor_id, direction_id, name, description,
		          duration_minutes, price, is_active, created_at, updated_at`,
		input.DoctorID, input.DirectionID, input.Name, input.Description,
		input.DurationMinutes, input.Price).
		Scan(&s.ID, &s.DoctorID, &s.DirectionID, &s.Name, &s.Description,
			&s.DurationMinutes, &s.Price, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepo) Update(ctx context.Context, id int64, input UpdateServiceInput) (*model.Service, error) {
	var s model.Service
	err := r.db.QueryRow(ctx, `
		UPDATE services
		SET    direction_id = $1, name = $2, description = $3,
		       duration_minutes = $4, price = $5, updated_at = NOW()
		WHERE  id = $6
		RETURNING id, doctor_id, direction_id, name, description,
		          duration_minutes, price, is_active, created_at, updated_at`,
		input.DirectionID, input.Name, input.Description,
		input.DurationMinutes, input.Price, id).
		Scan(&s.ID, &s.DoctorID, &s.DirectionID, &s.Name, &s.Description,
			&s.DurationMinutes, &s.Price, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepo) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE services SET is_active = false, updated_at = NOW()
		WHERE  id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// GetDurationMinutes satisfies availability.ServiceRepository.
func (r *ServiceRepo) GetDurationMinutes(ctx context.Context, serviceID int64) (int, error) {
	var duration int
	err := r.db.QueryRow(ctx,
		`SELECT duration_minutes FROM services WHERE id = $1 AND is_active = true`,
		serviceID).Scan(&duration)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, apperrors.ErrNotFound
		}
		return 0, err
	}
	return duration, nil
}
