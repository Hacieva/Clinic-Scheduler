package model

import "time"

// AppointmentStatus mirrors CHECK (status IN (...)) on the appointments table.
type AppointmentStatus string

const (
	StatusCreated             AppointmentStatus = "created"
	StatusConfirmed           AppointmentStatus = "confirmed"
	StatusCancelledByPatient  AppointmentStatus = "cancelled_by_patient"
	StatusCancelledByAdmin    AppointmentStatus = "cancelled_by_admin"
	StatusCompleted           AppointmentStatus = "completed"
	StatusNoShow              AppointmentStatus = "no_show"
)

// AppointmentSource mirrors CHECK (source IN (...)) on the appointments table.
type AppointmentSource string

const (
	SourceTelegramBot AppointmentSource = "telegram_bot"
	SourceAdminPanel  AppointmentSource = "admin_panel"
)

// Appointment mirrors the appointments table.
type Appointment struct {
	ID             int64             `json:"id"`
	PatientID      int64             `json:"patient_id"`
	DoctorID       int64             `json:"doctor_id"`
	ServiceID      int64             `json:"service_id"`
	DirectionID    *int64            `json:"direction_id,omitempty"`
	BranchID       *int64            `json:"branch_id,omitempty"`
	StartAt        time.Time         `json:"start_at"`
	EndAt          time.Time         `json:"end_at"`
	Status         AppointmentStatus `json:"status"`
	Source         AppointmentSource `json:"source"`
	PatientComment *string           `json:"patient_comment,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// AppointmentStatusHistory mirrors the appointment_status_history table.
type AppointmentStatusHistory struct {
	ID               int64     `json:"id"`
	AppointmentID    int64     `json:"appointment_id"`
	OldStatus        *string   `json:"old_status"`
	NewStatus        string    `json:"new_status"`
	ChangedByUserID  *int64    `json:"changed_by_user_id"`
	ChangedAt        time.Time `json:"changed_at"`
	Comment          *string   `json:"comment"`
}
