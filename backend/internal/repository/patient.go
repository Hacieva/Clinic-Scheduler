package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

type PatientRepository interface {
	List(ctx context.Context, filter PatientFilter) ([]model.Patient, error)
	GetByID(ctx context.Context, id int64) (*model.Patient, error)
	GetByPhone(ctx context.Context, phone string) (*model.Patient, error)
	Create(ctx context.Context, input CreatePatientInput) (*model.Patient, error)
	Update(ctx context.Context, id int64, input UpdatePatientInput) (*model.Patient, error)
}

type PatientFilter struct {
	Search *string // ILIKE match against full_name, phone, email
	Source *string // optional: 'admin_panel' | 'telegram_bot'
	Limit  int     // clamped to [1, 100] by service; default 50
	Offset int
}

type CreatePatientInput struct {
	FullName    string
	Phone       string
	Email       *string
	DateOfBirth *time.Time
	Comment     *string
	Source      string
}

type UpdatePatientInput struct {
	FullName    *string
	Phone       *string
	DateOfBirth *time.Time
	Email       *string
	Comment     *string
}

type PatientRepo struct {
	db *pgxpool.Pool
}

func NewPatientRepo(db *pgxpool.Pool) *PatientRepo {
	return &PatientRepo{db: db}
}

const patientCols = `id, telegram_user_id, telegram_username, full_name, phone,
       date_of_birth, email, comment, source, created_at, updated_at`

func scanPatient(p *model.Patient, scan func(...any) error) error {
	return scan(
		&p.ID, &p.TelegramUserID, &p.TelegramUsername,
		&p.FullName, &p.Phone,
		&p.DateOfBirth, &p.Email, &p.Comment, &p.Source,
		&p.CreatedAt, &p.UpdatedAt,
	)
}

func (r *PatientRepo) List(ctx context.Context, filter PatientFilter) ([]model.Patient, error) {
	// Build query dynamically to support optional source filter.
	// last_appointment_at is computed via correlated subquery — populated only in list responses.
	query := `
		SELECT p.id, p.telegram_user_id, p.telegram_username, p.full_name, p.phone,
		       p.date_of_birth, p.email, p.comment, p.source, p.created_at, p.updated_at,
		       (SELECT MAX(a.start_at) FROM appointments a WHERE a.patient_id = p.id) AS last_appointment_at
		FROM   patients p
		WHERE  ($1::text IS NULL
		        OR p.full_name ILIKE '%' || $1 || '%'
		        OR p.phone     ILIKE '%' || $1 || '%'
		        OR p.email     ILIKE '%' || $1 || '%')`

	args := []any{filter.Search}
	n := 2

	if filter.Source != nil {
		query += fmt.Sprintf(` AND p.source = $%d`, n)
		args = append(args, *filter.Source)
		n++
	}

	query += ` ORDER BY p.full_name ASC`
	query += fmt.Sprintf(` LIMIT $%s OFFSET $%s`, strconv.Itoa(n), strconv.Itoa(n+1))
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []model.Patient{}
	for rows.Next() {
		var p model.Patient
		if err := rows.Scan(
			&p.ID, &p.TelegramUserID, &p.TelegramUsername,
			&p.FullName, &p.Phone,
			&p.DateOfBirth, &p.Email, &p.Comment, &p.Source,
			&p.CreatedAt, &p.UpdatedAt,
			&p.LastAppointmentAt,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PatientRepo) GetByID(ctx context.Context, id int64) (*model.Patient, error) {
	var p model.Patient
	err := scanPatient(&p, r.db.QueryRow(ctx,
		`SELECT `+patientCols+` FROM patients WHERE id = $1`, id).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *PatientRepo) GetByPhone(ctx context.Context, phone string) (*model.Patient, error) {
	var p model.Patient
	err := scanPatient(&p, r.db.QueryRow(ctx,
		`SELECT `+patientCols+` FROM patients WHERE phone = $1 LIMIT 1`, phone).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *PatientRepo) Create(ctx context.Context, input CreatePatientInput) (*model.Patient, error) {
	var p model.Patient
	err := scanPatient(&p, r.db.QueryRow(ctx, `
		INSERT INTO patients (full_name, phone, email, date_of_birth, comment, source)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+patientCols,
		input.FullName, input.Phone, input.Email,
		input.DateOfBirth, input.Comment, input.Source).Scan)
	if err != nil {
		if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return nil, apperrors.ErrInvalidInput
		}
		return nil, err
	}
	return &p, nil
}

func (r *PatientRepo) Update(ctx context.Context, id int64, input UpdatePatientInput) (*model.Patient, error) {
	var p model.Patient
	err := scanPatient(&p, r.db.QueryRow(ctx, `
		UPDATE patients
		SET    full_name     = COALESCE($2, full_name),
		       phone         = COALESCE($3, phone),
		       date_of_birth = COALESCE($4, date_of_birth),
		       email         = COALESCE($5, email),
		       comment       = COALESCE($6, comment),
		       updated_at    = NOW()
		WHERE  id = $1
		RETURNING `+patientCols,
		id, input.FullName, input.Phone, input.DateOfBirth,
		input.Email, input.Comment).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return nil, apperrors.ErrInvalidInput
		}
		return nil, err
	}
	return &p, nil
}
