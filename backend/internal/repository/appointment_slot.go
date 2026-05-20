package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Hacieva/clinic-scheduler/backend/internal/availability"
)

// blockingStatuses are the only appointment statuses that occupy a doctor's time slot.
// Cancelled / completed / no-show appointments free the slot.
var blockingStatuses = []string{"created", "confirmed"}

// AppointmentSlotRepo satisfies availability.AppointmentRepository.
// It converts appointments rows into the plain availability.Slot type used by the calculator.
type AppointmentSlotRepo struct {
	db *pgxpool.Pool
}

func NewAppointmentSlotRepo(db *pgxpool.Pool) *AppointmentSlotRepo {
	return &AppointmentSlotRepo{db: db}
}

func (r *AppointmentSlotRepo) GetSlotsByDoctor(ctx context.Context, doctorID int64, from, to time.Time) ([]availability.Slot, error) {
	// Include the full last day: appointments that start before midnight of to+1.
	toExclusive := to.AddDate(0, 0, 1)

	rows, err := r.db.Query(ctx, `
		SELECT start_at, end_at
		FROM   appointments
		WHERE  doctor_id = $1
		  AND  start_at >= $2
		  AND  start_at < $3
		  AND  status = ANY($4)
		ORDER  BY start_at`,
		doctorID, from, toExclusive, blockingStatuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []availability.Slot
	for rows.Next() {
		var s availability.Slot
		if err := rows.Scan(&s.Start, &s.End); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}
