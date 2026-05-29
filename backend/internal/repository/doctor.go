package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

type DoctorRepository interface {
	List(ctx context.Context, filter DoctorFilter) ([]model.DoctorWithDirections, error)
	GetByID(ctx context.Context, id int64) (*model.DoctorWithDirections, error)
	GetDoctorIDByUserID(ctx context.Context, userID int64) (int64, error)
	Create(ctx context.Context, input CreateDoctorInput) (*model.Doctor, error)
	CreateWithAccount(ctx context.Context, input CreateDoctorInput, email, passwordHash string) (*model.Doctor, error)
	Update(ctx context.Context, id int64, input UpdateDoctorInput) (*model.Doctor, error)
	SoftDelete(ctx context.Context, id int64) error
	CreateAccount(ctx context.Context, doctorID int64, email, passwordHash string) (*model.Doctor, error)
	SetDirections(ctx context.Context, doctorID int64, directionIDs []int64) error
}

type CreateDoctorInput struct {
	FirstName   string
	LastName    string
	MiddleName  *string
	Cabinet     *string
	BranchID    *int64
	Phone       *string
	Description *string
	PhotoURL    *string
}

type UpdateDoctorInput = CreateDoctorInput

// DoctorFilter holds optional predicates for the List query.
type DoctorFilter struct {
	DirectionID *int64
	BranchID    *int64
}

type DoctorRepo struct {
	db *pgxpool.Pool
}

func NewDoctorRepo(db *pgxpool.Pool) *DoctorRepo {
	return &DoctorRepo{db: db}
}

// List returns all doctors, optionally filtered by direction and/or branch.
// Nil fields are ignored — existing behaviour is preserved when filter is zero-value.
func (r *DoctorRepo) List(ctx context.Context, filter DoctorFilter) ([]model.DoctorWithDirections, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.user_id, d.first_name, d.last_name, d.middle_name,
		       d.cabinet, d.branch_id, d.phone, d.description, d.photo_url,
		       d.is_active, d.created_at, d.updated_at, d.doctor_kind, d.booking_mode,
		       dir.id, dir.name, dir.description, dir.is_active, dir.created_at, dir.updated_at
		FROM   doctors d
		LEFT   JOIN doctor_directions dd ON dd.doctor_id = d.id
		LEFT   JOIN directions dir ON dir.id = dd.direction_id
		WHERE  ($1::bigint IS NULL OR EXISTS (
		           SELECT 1 FROM doctor_directions
		           WHERE  doctor_id = d.id AND direction_id = $1
		       ))
		  AND  ($2::bigint IS NULL OR d.branch_id = $2)
		ORDER  BY d.id, dir.id`, filter.DirectionID, filter.BranchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDoctorsWithDirections(rows)
}

func (r *DoctorRepo) GetByID(ctx context.Context, id int64) (*model.DoctorWithDirections, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.user_id, d.first_name, d.last_name, d.middle_name,
		       d.cabinet, d.branch_id, d.phone, d.description, d.photo_url,
		       d.is_active, d.created_at, d.updated_at, d.doctor_kind, d.booking_mode,
		       dir.id, dir.name, dir.description, dir.is_active, dir.created_at, dir.updated_at
		FROM   doctors d
		LEFT   JOIN doctor_directions dd ON dd.doctor_id = d.id
		LEFT   JOIN directions dir ON dir.id = dd.direction_id
		WHERE  d.id = $1
		ORDER  BY dir.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanDoctorsWithDirections(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, apperrors.ErrNotFound
	}
	return &results[0], nil
}

func (r *DoctorRepo) Create(ctx context.Context, input CreateDoctorInput) (*model.Doctor, error) {
	d := &model.Doctor{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO doctors (first_name, last_name, middle_name, cabinet, branch_id, phone, description, photo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, first_name, last_name, middle_name,
		          cabinet, branch_id, phone, description, photo_url,
		          is_active, created_at, updated_at`,
		input.FirstName, input.LastName, input.MiddleName,
		input.Cabinet, input.BranchID, input.Phone, input.Description, input.PhotoURL).
		Scan(&d.ID, &d.UserID, &d.FirstName, &d.LastName, &d.MiddleName,
			&d.Cabinet, &d.BranchID, &d.Phone, &d.Description, &d.PhotoURL,
			&d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// CreateWithAccount atomically creates a user account and a doctor linked to it.
func (r *DoctorRepo) CreateWithAccount(ctx context.Context, input CreateDoctorInput, email, passwordHash string) (*model.Doctor, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var newUserID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, is_active)
		VALUES ($1, $2, 'doctor', true)
		RETURNING id`, email, passwordHash).Scan(&newUserID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}

	d := &model.Doctor{}
	err = tx.QueryRow(ctx, `
		INSERT INTO doctors (user_id, first_name, last_name, middle_name, cabinet, branch_id, phone, description, photo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, first_name, last_name, middle_name,
		          cabinet, branch_id, phone, description, photo_url,
		          is_active, created_at, updated_at`,
		newUserID, input.FirstName, input.LastName, input.MiddleName,
		input.Cabinet, input.BranchID, input.Phone, input.Description, input.PhotoURL).
		Scan(&d.ID, &d.UserID, &d.FirstName, &d.LastName, &d.MiddleName,
			&d.Cabinet, &d.BranchID, &d.Phone, &d.Description, &d.PhotoURL,
			&d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DoctorRepo) Update(ctx context.Context, id int64, input UpdateDoctorInput) (*model.Doctor, error) {
	d := &model.Doctor{}
	err := r.db.QueryRow(ctx, `
		UPDATE doctors
		SET    first_name = $1, last_name = $2, middle_name = $3,
		       cabinet = $4, branch_id = $5, phone = $6, description = $7,
		       photo_url = $8, updated_at = NOW()
		WHERE  id = $9
		RETURNING id, user_id, first_name, last_name, middle_name,
		          cabinet, branch_id, phone, description, photo_url,
		          is_active, created_at, updated_at`,
		input.FirstName, input.LastName, input.MiddleName,
		input.Cabinet, input.BranchID, input.Phone, input.Description, input.PhotoURL, id).
		Scan(&d.ID, &d.UserID, &d.FirstName, &d.LastName, &d.MiddleName,
			&d.Cabinet, &d.BranchID, &d.Phone, &d.Description, &d.PhotoURL,
			&d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

func (r *DoctorRepo) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE doctors SET is_active = false, updated_at = NOW()
		WHERE  id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// CreateAccount atomically creates a user account and links it to the doctor.
// Returns ErrNotFound if the doctor does not exist,
// ErrAccountExists if the doctor already has a linked account,
// ErrConflict if the email is already taken.
func (r *DoctorRepo) CreateAccount(ctx context.Context, doctorID int64, email, passwordHash string) (*model.Doctor, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var existingUserID *int64
	err = tx.QueryRow(ctx,
		`SELECT user_id FROM doctors WHERE id = $1 FOR UPDATE`, doctorID).
		Scan(&existingUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	if existingUserID != nil {
		return nil, apperrors.ErrAccountExists
	}

	var newUserID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, is_active)
		VALUES ($1, $2, 'doctor', true)
		RETURNING id`, email, passwordHash).
		Scan(&newUserID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}

	d := &model.Doctor{}
	err = tx.QueryRow(ctx, `
		UPDATE doctors SET user_id = $1, updated_at = NOW()
		WHERE  id = $2
		RETURNING id, user_id, first_name, last_name, middle_name,
		          cabinet, branch_id, phone, description, photo_url,
		          is_active, created_at, updated_at`,
		newUserID, doctorID).
		Scan(&d.ID, &d.UserID, &d.FirstName, &d.LastName, &d.MiddleName,
			&d.Cabinet, &d.BranchID, &d.Phone, &d.Description, &d.PhotoURL,
			&d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// GetDoctorIDByUserID returns the doctor's primary key for a given user account.
// Used to resolve a JWT user_id to doctor_id for doctor-facing endpoints.
func (r *DoctorRepo) GetDoctorIDByUserID(ctx context.Context, userID int64) (int64, error) {
	var doctorID int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM doctors WHERE user_id = $1`, userID).Scan(&doctorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, apperrors.ErrNotFound
		}
		return 0, err
	}
	return doctorID, nil
}

// SetDirections atomically replaces all directions assigned to the doctor.
func (r *DoctorRepo) SetDirections(ctx context.Context, doctorID int64, directionIDs []int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM doctor_directions WHERE doctor_id = $1`, doctorID); err != nil {
		return err
	}

	for _, dirID := range directionIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO doctor_directions (doctor_id, direction_id)
			VALUES ($1, $2)`, doctorID, dirID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// scanDoctorsWithDirections groups JOIN rows by doctor, collecting directions per doctor.
func scanDoctorsWithDirections(rows pgx.Rows) ([]model.DoctorWithDirections, error) {
	index := make(map[int64]int)
	var result []model.DoctorWithDirections

	for rows.Next() {
		var doc model.Doctor
		var dirID       *int64
		var dirName     *string
		var dirDesc     *string
		var dirIsActive *bool
		var dirCreatedAt *time.Time
		var dirUpdatedAt *time.Time

		if err := rows.Scan(
			&doc.ID, &doc.UserID, &doc.FirstName, &doc.LastName, &doc.MiddleName,
			&doc.Cabinet, &doc.BranchID, &doc.Phone, &doc.Description, &doc.PhotoURL,
			&doc.IsActive, &doc.CreatedAt, &doc.UpdatedAt, &doc.DoctorKind, &doc.BookingMode,
			&dirID, &dirName, &dirDesc, &dirIsActive, &dirCreatedAt, &dirUpdatedAt,
		); err != nil {
			return nil, err
		}

		idx, exists := index[doc.ID]
		if !exists {
			idx = len(result)
			index[doc.ID] = idx
			result = append(result, model.DoctorWithDirections{
				Doctor:     doc,
				Directions: []model.Direction{},
			})
		}

		if dirID != nil {
			result[idx].Directions = append(result[idx].Directions, model.Direction{
				ID:          *dirID,
				Name:        *dirName,
				Description: dirDesc,
				IsActive:    *dirIsActive,
				CreatedAt:   *dirCreatedAt,
				UpdatedAt:   *dirUpdatedAt,
			})
		}
	}

	if result == nil {
		result = []model.DoctorWithDirections{}
	}
	return result, rows.Err()
}
