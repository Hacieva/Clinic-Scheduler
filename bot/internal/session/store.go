package session

import "context"

// Store is the session persistence contract for the Telegram booking FSM.
// All operations are keyed by telegram_user_id.
type Store interface {
	// Get returns nil, nil when no session exists for the user.
	Get(ctx context.Context, telegramUserID int64) (*Data, error)
	// Replace atomically creates or overwrites the session (UPSERT).
	Replace(ctx context.Context, telegramUserID int64, data *Data) error
	// Delete removes the session. Safe to call when no session exists.
	Delete(ctx context.Context, telegramUserID int64) error
}

// Data is the minimal FSM payload persisted per Telegram user.
// Display names (DirectionName, DoctorName, ServiceName) are kept alongside
// IDs to avoid a re-fetch when rendering the confirmation screen.
// No JWT, no bot token, no full entity objects, no availability lists.
type Data struct {
	State         string `json:"state"`
	DirectionID   *int64 `json:"direction_id,omitempty"`
	DirectionName string `json:"direction_name,omitempty"`
	DoctorID      *int64 `json:"doctor_id,omitempty"`
	DoctorName    string `json:"doctor_name,omitempty"`
	ServiceID     *int64 `json:"service_id,omitempty"`
	ServiceName   string `json:"service_name,omitempty"`
	ServicePrice  *int64 `json:"service_price,omitempty"` // kopecks
	Date          string `json:"date,omitempty"`          // YYYY-MM-DD
	Time          string `json:"time,omitempty"`          // HH:MM
	PatientName   string `json:"patient_name,omitempty"`
	PatientPhone  string `json:"patient_phone,omitempty"`
}
