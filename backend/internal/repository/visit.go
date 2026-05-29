package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

// VisitRepository covers all visit persistence operations.
type VisitRepository interface {
	Create(ctx context.Context, input CreateVisitInput) (*model.Visit, error)
	GetByID(ctx context.Context, id int64) (*model.Visit, error)
	List(ctx context.Context, filter VisitFilter) ([]model.Visit, error)
	UpdateStatus(ctx context.Context, id int64, status model.VisitStatus, arrivedAt, completedAt *time.Time) error
	UpdatePatientID(ctx context.Context, id int64, patientID int64) error
}

// CreateVisitInput carries the data needed to create a new visit.
type CreateVisitInput struct {
	PatientID int64
	BranchID  int64
	VisitType model.VisitType
	Status    model.VisitStatus
	ArrivedAt *time.Time
	Comment   *string
}

// VisitFilter defines optional predicates and pagination for List.
type VisitFilter struct {
	PatientID *int64
	BranchID  *int64
	Status    *model.VisitStatus
	DateFrom  *time.Time
	DateTo    *time.Time
	Limit     int
	Offset    int
}

type VisitRepo struct {
	db *pgxpool.Pool
}

func NewVisitRepo(db *pgxpool.Pool) *VisitRepo {
	return &VisitRepo{db: db}
}

const visitCols = `id, patient_id, branch_id, visit_type, status,
       arrived_at, completed_at, comment, created_at, updated_at`

func scanVisit(v *model.Visit, scan func(...any) error) error {
	return scan(
		&v.ID, &v.PatientID, &v.BranchID, &v.VisitType, &v.Status,
		&v.ArrivedAt, &v.CompletedAt, &v.Comment,
		&v.CreatedAt, &v.UpdatedAt,
	)
}

func (r *VisitRepo) Create(ctx context.Context, input CreateVisitInput) (*model.Visit, error) {
	status := input.Status
	if status == "" {
		status = model.VisitStatusScheduled
	}
	var v model.Visit
	err := scanVisit(&v, r.db.QueryRow(ctx, `
		INSERT INTO visits (patient_id, branch_id, visit_type, status, arrived_at, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+visitCols,
		input.PatientID, input.BranchID, string(input.VisitType),
		string(status), input.ArrivedAt, input.Comment).Scan)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VisitRepo) GetByID(ctx context.Context, id int64) (*model.Visit, error) {
	var v model.Visit
	err := scanVisit(&v, r.db.QueryRow(ctx,
		`SELECT `+visitCols+` FROM visits WHERE id = $1`, id).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

func (r *VisitRepo) List(ctx context.Context, filter VisitFilter) ([]model.Visit, error) {
	query := `SELECT ` + visitCols + ` FROM visits WHERE 1=1`
	args := []any{}
	n := 1

	if filter.PatientID != nil {
		query += fmt.Sprintf(` AND patient_id = $%d`, n)
		args = append(args, *filter.PatientID)
		n++
	}
	if filter.BranchID != nil {
		query += fmt.Sprintf(` AND branch_id = $%d`, n)
		args = append(args, *filter.BranchID)
		n++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(` AND status = $%d`, n)
		args = append(args, string(*filter.Status))
		n++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(` AND created_at >= $%d`, n)
		args = append(args, *filter.DateFrom)
		n++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(` AND created_at < $%d`, n)
		args = append(args, filter.DateTo.AddDate(0, 0, 1))
		n++
	}

	query += ` ORDER BY created_at DESC`

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, n, n+1)
	args = append(args, limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.Visit{}
	for rows.Next() {
		var v model.Visit
		if err := scanVisit(&v, rows.Scan); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *VisitRepo) UpdatePatientID(ctx context.Context, id int64, patientID int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE visits SET patient_id = $2, updated_at = NOW() WHERE id = $1`,
		id, patientID)
	return err
}

func (r *VisitRepo) UpdateStatus(
	ctx context.Context,
	id int64,
	status model.VisitStatus,
	arrivedAt, completedAt *time.Time,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE visits
		SET    status       = $2,
		       arrived_at   = COALESCE($3, arrived_at),
		       completed_at = COALESCE($4, completed_at),
		       updated_at   = NOW()
		WHERE  id = $1`,
		id, string(status), arrivedAt, completedAt)
	return err
}
