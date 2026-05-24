package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

// DoctorServiceRepository manages the doctor_services junction table.
// This is the authoritative source for doctor–service assignments.
type DoctorServiceRepository interface {
	ListAssignedToDoctor(ctx context.Context, doctorID int64) ([]model.Service, error)
	IsAssigned(ctx context.Context, doctorID, serviceID int64) (bool, error)
	Assign(ctx context.Context, doctorID, serviceID int64) error
	Unassign(ctx context.Context, doctorID, serviceID int64) error
	BulkReplace(ctx context.Context, doctorID int64, serviceIDs []int64) error
}

type DoctorServiceRepo struct {
	db *pgxpool.Pool
}

func NewDoctorServiceRepo(db *pgxpool.Pool) *DoctorServiceRepo {
	return &DoctorServiceRepo{db: db}
}

func (r *DoctorServiceRepo) ListAssignedToDoctor(ctx context.Context, doctorID int64) ([]model.Service, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.doctor_id, s.direction_id, s.category, s.name, s.description,
		       s.duration_minutes, s.price, s.is_active, s.created_at, s.updated_at
		FROM   services s
		JOIN   doctor_services ds ON ds.service_id = s.id
		WHERE  ds.doctor_id = $1 AND s.is_active = true
		ORDER  BY s.name`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Service
	for rows.Next() {
		var s model.Service
		if err := rows.Scan(
			&s.ID, &s.DoctorID, &s.DirectionID, &s.Category, &s.Name, &s.Description,
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

func (r *DoctorServiceRepo) IsAssigned(ctx context.Context, doctorID, serviceID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM doctor_services
			WHERE doctor_id = $1 AND service_id = $2
		)`, doctorID, serviceID).Scan(&exists)
	return exists, err
}

func (r *DoctorServiceRepo) Assign(ctx context.Context, doctorID, serviceID int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO doctor_services (doctor_id, service_id)
		VALUES ($1, $2)
		ON CONFLICT (doctor_id, service_id) DO NOTHING`, doctorID, serviceID)
	return err
}

func (r *DoctorServiceRepo) Unassign(ctx context.Context, doctorID, serviceID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM doctor_services WHERE doctor_id = $1 AND service_id = $2`,
		doctorID, serviceID)
	return err
}

func (r *DoctorServiceRepo) BulkReplace(ctx context.Context, doctorID int64, serviceIDs []int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM doctor_services WHERE doctor_id = $1`, doctorID); err != nil {
		return err
	}

	for _, svcID := range serviceIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO doctor_services (doctor_id, service_id)
			VALUES ($1, $2)
			ON CONFLICT (doctor_id, service_id) DO NOTHING`, doctorID, svcID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
