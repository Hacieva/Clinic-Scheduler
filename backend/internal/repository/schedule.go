package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Hacieva/clinic-scheduler/backend/internal/availability"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

// ScheduleRepository covers management CRUD for working hours and exceptions.
// The concrete ScheduleRepo also satisfies availability.ScheduleRepository (duck typing).
type ScheduleRepository interface {
	ListWorkingHours(ctx context.Context, doctorID int64) ([]model.WorkingHours, error)
	ReplaceWorkingHours(ctx context.Context, doctorID int64, inputs []CreateWorkingHoursInput) error
	ListExceptions(ctx context.Context, doctorID int64, from, to time.Time) ([]model.ScheduleException, error)
	CreateException(ctx context.Context, input CreateExceptionInput) (*model.ScheduleException, error)
	UpdateException(ctx context.Context, id int64, input CreateExceptionInput) (*model.ScheduleException, error)
	DeleteException(ctx context.Context, id int64) error
}

type CreateWorkingHoursInput struct {
	DayOfWeek int       // 1–7 (1=Mon, 7=Sun)
	StartTime time.Time // time-of-day only
	EndTime   time.Time // time-of-day only
}

type CreateExceptionInput struct {
	DoctorID  int64
	Date      time.Time
	Type      model.ExceptionType
	StartTime *time.Time
	EndTime   *time.Time
	Comment   *string
}

type ScheduleRepo struct {
	db *pgxpool.Pool
}

func NewScheduleRepo(db *pgxpool.Pool) *ScheduleRepo {
	return &ScheduleRepo{db: db}
}

func (r *ScheduleRepo) ListWorkingHours(ctx context.Context, doctorID int64) ([]model.WorkingHours, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, doctor_id, day_of_week, start_time, end_time, is_active, created_at, updated_at
		FROM   doctor_working_hours
		WHERE  doctor_id = $1
		ORDER  BY day_of_week, start_time`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.WorkingHours
	for rows.Next() {
		var wh model.WorkingHours
		if err := rows.Scan(&wh.ID, &wh.DoctorID, &wh.DayOfWeek, &wh.StartTime, &wh.EndTime,
			&wh.IsActive, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, wh)
	}
	if result == nil {
		result = []model.WorkingHours{}
	}
	return result, rows.Err()
}

// ReplaceWorkingHours atomically removes all existing working hours for the doctor
// and inserts the new set inside a single transaction.
func (r *ScheduleRepo) ReplaceWorkingHours(ctx context.Context, doctorID int64, inputs []CreateWorkingHoursInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM doctor_working_hours WHERE doctor_id = $1`, doctorID); err != nil {
		return err
	}

	for _, inp := range inputs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO doctor_working_hours (doctor_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4)`,
			doctorID, inp.DayOfWeek, inp.StartTime, inp.EndTime); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ScheduleRepo) ListExceptions(ctx context.Context, doctorID int64, from, to time.Time) ([]model.ScheduleException, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, doctor_id, date, type, start_time, end_time, comment, created_at, updated_at
		FROM   doctor_schedule_exceptions
		WHERE  doctor_id = $1 AND date BETWEEN $2 AND $3
		ORDER  BY date`, doctorID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.ScheduleException
	for rows.Next() {
		var ex model.ScheduleException
		if err := rows.Scan(&ex.ID, &ex.DoctorID, &ex.Date, &ex.Type,
			&ex.StartTime, &ex.EndTime, &ex.Comment, &ex.CreatedAt, &ex.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, ex)
	}
	if result == nil {
		result = []model.ScheduleException{}
	}
	return result, rows.Err()
}

func (r *ScheduleRepo) CreateException(ctx context.Context, input CreateExceptionInput) (*model.ScheduleException, error) {
	var ex model.ScheduleException
	err := r.db.QueryRow(ctx, `
		INSERT INTO doctor_schedule_exceptions (doctor_id, date, type, start_time, end_time, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, doctor_id, date, type, start_time, end_time, comment, created_at, updated_at`,
		input.DoctorID, input.Date, input.Type, input.StartTime, input.EndTime, input.Comment).
		Scan(&ex.ID, &ex.DoctorID, &ex.Date, &ex.Type,
			&ex.StartTime, &ex.EndTime, &ex.Comment, &ex.CreatedAt, &ex.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &ex, nil
}

func (r *ScheduleRepo) UpdateException(ctx context.Context, id int64, input CreateExceptionInput) (*model.ScheduleException, error) {
	var ex model.ScheduleException
	err := r.db.QueryRow(ctx, `
		UPDATE doctor_schedule_exceptions
		SET    date = $1, type = $2, start_time = $3, end_time = $4, comment = $5, updated_at = NOW()
		WHERE  id = $6
		RETURNING id, doctor_id, date, type, start_time, end_time, comment, created_at, updated_at`,
		input.Date, input.Type, input.StartTime, input.EndTime, input.Comment, id).
		Scan(&ex.ID, &ex.DoctorID, &ex.Date, &ex.Type,
			&ex.StartTime, &ex.EndTime, &ex.Comment, &ex.CreatedAt, &ex.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &ex, nil
}

func (r *ScheduleRepo) DeleteException(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM doctor_schedule_exceptions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// GetWorkingHours satisfies availability.ScheduleRepository.
// Converts DB day_of_week (1–7) to time.Weekday.
func (r *ScheduleRepo) GetWorkingHours(ctx context.Context, doctorID int64) ([]availability.RegularSchedule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT day_of_week, start_time, end_time
		FROM   doctor_working_hours
		WHERE  doctor_id = $1 AND is_active = true
		ORDER  BY day_of_week, start_time`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []availability.RegularSchedule
	for rows.Next() {
		var dbDay int
		var start, end time.Time
		if err := rows.Scan(&dbDay, &start, &end); err != nil {
			return nil, err
		}
		result = append(result, availability.RegularSchedule{
			DayOfWeek: dbDayToWeekday(dbDay),
			Start:     start,
			End:       end,
		})
	}
	return result, rows.Err()
}

// GetScheduleExceptions satisfies availability.ScheduleRepository.
func (r *ScheduleRepo) GetScheduleExceptions(ctx context.Context, doctorID int64, from, to time.Time) ([]availability.Exception, error) {
	rows, err := r.db.Query(ctx, `
		SELECT date, type, start_time, end_time
		FROM   doctor_schedule_exceptions
		WHERE  doctor_id = $1 AND date BETWEEN $2 AND $3
		ORDER  BY date`, doctorID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []availability.Exception
	for rows.Next() {
		var ex availability.Exception
		if err := rows.Scan(&ex.Date, &ex.Type, &ex.Start, &ex.End); err != nil {
			return nil, err
		}
		result = append(result, ex)
	}
	return result, rows.Err()
}

// dbDayToWeekday converts DB day_of_week (1=Mon … 7=Sun) to time.Weekday.
// Uses modulo: DB 7 → 7%7=0 = time.Sunday.
func dbDayToWeekday(db int) time.Weekday {
	return time.Weekday(db % 7)
}

// weekdayToDBDay converts time.Weekday to DB day_of_week (1=Mon … 7=Sun).
func weekdayToDBDay(wd time.Weekday) int {
	if wd == time.Sunday {
		return 7
	}
	return int(wd)
}
