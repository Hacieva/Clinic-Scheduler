package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

// AppointmentRepository covers all appointment persistence operations.
type AppointmentRepository interface {
	Create(ctx context.Context, input CreateAppointmentInput) (*model.Appointment, error)
	GetByID(ctx context.Context, id int64) (*AppointmentDetail, error)
	List(ctx context.Context, filter AppointmentFilter) ([]AppointmentDetail, error)
	UpdateStatus(ctx context.Context, id int64, fromStatus, newStatus model.AppointmentStatus, changedByUserID *int64, comment *string) error
}

// CreateAppointmentInput carries all data needed to atomically create an appointment.
// EndAt and DirectionID are computed by the service layer before calling Create.
type CreateAppointmentInput struct {
	// PatientID, when set, skips the patient upsert step in the transaction.
	// Used by walk-in flow where patient is resolved in the service layer first.
	PatientID               *int64
	PatientTelegramID       *int64
	PatientTelegramUsername *string
	PatientName             string
	PatientPhone            string
	DoctorID                int64
	ServiceID               int64
	DirectionID             *int64
	BranchID                *int64
	VisitID                 *int64          // if nil, auto-created for scheduled; must be set for walk-in
	AppointmentType         model.AppointmentType
	StartAt                 time.Time
	EndAt                   time.Time
	Source                  model.AppointmentSource
	PatientComment          *string
	CreatedByUserID         *int64
}

// AppointmentDetail is an appointment row joined with patient, doctor, service, and branch names.
// Handler layer is responsible for omitting sensitive patient fields in doctor-facing responses.
type AppointmentDetail struct {
	model.Appointment
	PatientName       string  `json:"patient_name"`
	PatientPhone      string  `json:"patient_phone"`
	PatientTelegramID *int64  `json:"patient_telegram_id,omitempty"`
	DoctorFullName    string  `json:"doctor_full_name"`
	ServiceName       string  `json:"service_name"`
	BranchName        *string `json:"branch_name,omitempty"`
}

// AppointmentFilter defines optional predicates and pagination for List.
// Limit is clamped to [1, 100]; default 50.
type AppointmentFilter struct {
	DoctorID  *int64
	PatientID *int64
	VisitID   *int64
	BranchID  *int64
	Status    *model.AppointmentStatus
	DateFrom  *time.Time
	DateTo    *time.Time
	Limit     int
	Offset    int
}

type AppointmentRepo struct {
	db *pgxpool.Pool
}

func NewAppointmentRepo(db *pgxpool.Pool) *AppointmentRepo {
	return &AppointmentRepo{db: db}
}

// Create runs a full atomic transaction:
//  1. Pessimistic conflict check (scheduled only) via SELECT … FOR UPDATE
//  2. Patient upsert (telegram path) or plain INSERT (admin-panel path)
//  3. [Optional] Validate VisitID belongs to the same patient
//  4. INSERT appointment (with visit_id, appointment_type, branch_id)
//  5. INSERT initial status-history entry
//
// The EXCLUDE USING GIST constraint is a second line of defence for scheduled
// appointments — it catches any concurrent insertion that slips past the FOR UPDATE.
// Walk-in appointments bypass slot conflict checks (they enter a queue, not a slot).
func (r *AppointmentRepo) Create(ctx context.Context, input CreateAppointmentInput) (*model.Appointment, error) {
	apptType := input.AppointmentType
	if apptType == "" {
		apptType = model.AppointmentTypeScheduled
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Lock any overlapping active appointment for the same doctor.
	//    Walk-in appointments have no reserved slot — skip this check.
	if apptType == model.AppointmentTypeScheduled {
		var conflictID int64
		err = tx.QueryRow(ctx, `
			SELECT id FROM appointments
			WHERE  doctor_id = $1
			  AND  appointment_type = 'scheduled'
			  AND  status IN ('created', 'confirmed', 'arrived')
			  AND  tstzrange(start_at, end_at) && tstzrange($2, $3)
			ORDER  BY id
			LIMIT  1
			FOR UPDATE`,
			input.DoctorID, input.StartAt, input.EndAt).Scan(&conflictID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			return nil, apperrors.ErrSlotTaken
		}
	}

	// 2. Patient persistence.
	//    If PatientID is pre-resolved (walk-in flow), skip patient upsert entirely.
	//    When telegram_user_id IS NOT NULL we upsert — the patient may have booked before.
	//    When it IS NULL (admin-panel booking) we do a plain INSERT.
	var patientID int64
	if input.PatientID != nil {
		patientID = *input.PatientID
	} else if input.PatientTelegramID != nil {
		err = tx.QueryRow(ctx, `
			INSERT INTO patients (telegram_user_id, telegram_username, full_name, phone)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (telegram_user_id) DO UPDATE
			  SET telegram_username = EXCLUDED.telegram_username,
			      full_name         = EXCLUDED.full_name,
			      phone             = EXCLUDED.phone,
			      updated_at        = NOW()
			RETURNING id`,
			input.PatientTelegramID, input.PatientTelegramUsername,
			input.PatientName, input.PatientPhone).Scan(&patientID)
		if err != nil {
			return nil, err
		}
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO patients (full_name, phone)
			VALUES ($1, $2)
			RETURNING id`,
			input.PatientName, input.PatientPhone).Scan(&patientID)
		if err != nil {
			return nil, err
		}
	}

	// 3. Resolve visit_id: validate a caller-supplied value, or auto-create a Visit
	//    for scheduled appointments.
	//
	//    Both the Visit INSERT and the Appointment INSERT share this transaction.
	//    If anything below fails the whole tx rolls back — no orphaned visits are
	//    possible regardless of which step fails.
	var resolvedVisitID *int64
	if input.VisitID != nil {
		// Validate that the provided Visit belongs to the same patient.
		var vPatientID int64
		err = tx.QueryRow(ctx,
			`SELECT patient_id FROM visits WHERE id = $1`, *input.VisitID).Scan(&vPatientID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if vPatientID != patientID {
			return nil, apperrors.ErrVisitPatientMismatch
		}
		resolvedVisitID = input.VisitID
	} else if apptType == model.AppointmentTypeScheduled && input.BranchID != nil {
		// Auto-create a Visit for this scheduled appointment.
		// status = 'scheduled', visit_type = 'scheduled'.
		var vid int64
		err = tx.QueryRow(ctx, `
			INSERT INTO visits (patient_id, branch_id, visit_type, status)
			VALUES ($1, $2, 'scheduled', 'scheduled')
			RETURNING id`,
			patientID, *input.BranchID).Scan(&vid)
		if err != nil {
			return nil, err
		}
		resolvedVisitID = &vid
	}

	// 4. Determine initial status: walk-in starts as 'arrived'; scheduled starts as 'created'.
	initialStatus := model.StatusCreated
	if apptType == model.AppointmentTypeWalkIn {
		initialStatus = model.StatusArrived
	}

	// 5. Insert appointment.
	var appt model.Appointment
	err = tx.QueryRow(ctx, `
		INSERT INTO appointments
		       (patient_id, doctor_id, service_id, direction_id, branch_id,
		        visit_id, appointment_type,
		        start_at, end_at, status, source, patient_comment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, visit_id, patient_id, doctor_id, service_id, direction_id, branch_id,
		          appointment_type,
		          start_at, end_at, status, source, patient_comment,
		          created_at, updated_at`,
		patientID, input.DoctorID, input.ServiceID, input.DirectionID, input.BranchID,
		resolvedVisitID, string(apptType),
		input.StartAt, input.EndAt, string(initialStatus), input.Source, input.PatientComment).
		Scan(&appt.ID, &appt.VisitID, &appt.PatientID, &appt.DoctorID, &appt.ServiceID,
			&appt.DirectionID, &appt.BranchID, &appt.AppointmentType,
			&appt.StartAt, &appt.EndAt, &appt.Status, &appt.Source, &appt.PatientComment,
			&appt.CreatedAt, &appt.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23P01": // exclusion_violation: concurrent INSERT slipped past FOR UPDATE
				return nil, apperrors.ErrSlotTaken
			case "23503": // foreign_key_violation
				return nil, apperrors.ErrNotFound
			}
		}
		return nil, err
	}

	// 6. Initial status-history entry.
	if _, err = tx.Exec(ctx, `
		INSERT INTO appointment_status_history
		       (appointment_id, old_status, new_status, changed_by_user_id)
		VALUES ($1, NULL, $2, $3)`,
		appt.ID, string(initialStatus), input.CreatedByUserID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &appt, nil
}

// appointmentSelectBase is the common SELECT … FROM … JOIN fragment shared by GetByID and List.
const appointmentSelectBase = `
	SELECT a.id, a.visit_id, a.patient_id, a.doctor_id, a.service_id, a.direction_id,
	       a.branch_id, a.appointment_type,
	       a.start_at, a.end_at, a.status, a.source, a.patient_comment,
	       a.created_at, a.updated_at,
	       p.full_name, p.phone, p.telegram_user_id,
	       d.first_name || ' ' || d.last_name,
	       s.name,
	       b.name
	FROM   appointments a
	JOIN   patients  p ON p.id = a.patient_id
	JOIN   doctors   d ON d.id = a.doctor_id
	JOIN   services  s ON s.id = a.service_id
	LEFT JOIN branches b ON b.id = a.branch_id`

func scanAppointmentDetail(d *AppointmentDetail, scan func(...any) error) error {
	return scan(
		&d.ID, &d.VisitID, &d.PatientID, &d.DoctorID, &d.ServiceID, &d.DirectionID,
		&d.BranchID, &d.AppointmentType,
		&d.StartAt, &d.EndAt, &d.Status, &d.Source, &d.PatientComment,
		&d.CreatedAt, &d.UpdatedAt,
		&d.PatientName, &d.PatientPhone, &d.PatientTelegramID,
		&d.DoctorFullName, &d.ServiceName,
		&d.BranchName,
	)
}

func (r *AppointmentRepo) GetByID(ctx context.Context, id int64) (*AppointmentDetail, error) {
	var d AppointmentDetail
	err := scanAppointmentDetail(&d, r.db.QueryRow(ctx,
		appointmentSelectBase+` WHERE a.id = $1`, id).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

// List returns appointments matching the filter with deterministic ordering
// (start_at ASC, id ASC) to support stable pagination.
func (r *AppointmentRepo) List(ctx context.Context, filter AppointmentFilter) ([]AppointmentDetail, error) {
	query := appointmentSelectBase + ` WHERE 1=1`
	args := []any{}
	n := 1

	if filter.DoctorID != nil {
		query += fmt.Sprintf(` AND a.doctor_id = $%d`, n)
		args = append(args, *filter.DoctorID)
		n++
	}
	if filter.PatientID != nil {
		query += fmt.Sprintf(` AND a.patient_id = $%d`, n)
		args = append(args, *filter.PatientID)
		n++
	}
	if filter.VisitID != nil {
		query += fmt.Sprintf(` AND a.visit_id = $%d`, n)
		args = append(args, *filter.VisitID)
		n++
	}
	if filter.BranchID != nil {
		query += fmt.Sprintf(` AND a.branch_id = $%d`, n)
		args = append(args, *filter.BranchID)
		n++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(` AND a.status = $%d`, n)
		args = append(args, string(*filter.Status))
		n++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(` AND a.start_at >= $%d`, n)
		args = append(args, *filter.DateFrom)
		n++
	}
	if filter.DateTo != nil {
		// Include the full last day by shifting to start of next day.
		query += fmt.Sprintf(` AND a.start_at < $%d`, n)
		args = append(args, filter.DateTo.AddDate(0, 0, 1))
		n++
	}

	query += ` ORDER BY a.start_at ASC, a.id ASC`

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

	var result []AppointmentDetail
	for rows.Next() {
		var d AppointmentDetail
		if err := scanAppointmentDetail(&d, rows.Scan); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if result == nil {
		result = []AppointmentDetail{}
	}
	return result, rows.Err()
}

// UpdateStatus atomically reads the current status (FOR UPDATE), updates it, and
// appends a status-history entry — all within a single transaction.
// The service layer is responsible for validating the transition before calling this.
func (r *AppointmentRepo) UpdateStatus(
	ctx context.Context,
	id int64,
	fromStatus, newStatus model.AppointmentStatus,
	changedByUserID *int64,
	comment *string,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var oldStatus model.AppointmentStatus
	err = tx.QueryRow(ctx,
		`SELECT status FROM appointments WHERE id = $1 FOR UPDATE`, id).
		Scan(&oldStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.ErrNotFound
		}
		return err
	}

	// Re-validate the transition inside the lock: a concurrent status change between
	// the service-layer pre-check and this transaction would otherwise go undetected.
	if oldStatus != fromStatus {
		return apperrors.ErrInvalidStatusTransition
	}

	if _, err = tx.Exec(ctx,
		`UPDATE appointments SET status = $1, updated_at = NOW() WHERE id = $2`,
		newStatus, id); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO appointment_status_history
		       (appointment_id, old_status, new_status, changed_by_user_id, comment)
		VALUES ($1, $2, $3, $4, $5)`,
		id, string(oldStatus), string(newStatus), changedByUserID, comment); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
