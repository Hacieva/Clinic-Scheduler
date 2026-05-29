package model

import "time"

// VisitStatus mirrors CHECK (status IN (...)) on the visits table.
type VisitStatus string

const (
	VisitStatusScheduled  VisitStatus = "scheduled"
	VisitStatusInProgress VisitStatus = "in_progress"
	VisitStatusCompleted  VisitStatus = "completed"
	VisitStatusCancelled  VisitStatus = "cancelled"
	VisitStatusNoShow     VisitStatus = "no_show"
)

// VisitType mirrors CHECK (visit_type IN (...)) on the visits table.
type VisitType string

const (
	VisitTypeScheduled VisitType = "scheduled"
	VisitTypeWalkIn    VisitType = "walk_in"
)

// Visit represents one physical clinic case.
// A visit groups one or more appointments for the same patient on the same occasion.
// Scheduled visits are created when the first appointment is booked.
// Walk-in visits are created when the patient arrives without a prior booking.
type Visit struct {
	ID          int64       `json:"id"`
	PatientID   int64       `json:"patient_id"`
	BranchID    int64       `json:"branch_id"`
	VisitType   VisitType   `json:"visit_type"`
	Status      VisitStatus `json:"status"`
	ArrivedAt   *time.Time  `json:"arrived_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Comment     *string     `json:"comment,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
