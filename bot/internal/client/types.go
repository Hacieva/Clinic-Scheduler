package client

// Minimal DTOs used by the Telegram booking flow.
// Fields are a strict subset of backend model fields; extra fields are ignored on decode.

// Direction is what the keyboard builder needs for the direction step.
type Direction struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Doctor carries only what the keyboard builder needs to display and route.
// Name composition (first+last+middle) is presentation logic handled in keyboard/.
type Doctor struct {
	ID         int64   `json:"id"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	MiddleName *string `json:"middle_name"`
}

// Service carries the fields needed for slot selection and the confirmation screen.
type Service struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	DurationMinutes int    `json:"duration_minutes"`
	Price           *int64 `json:"price"` // kopecks; nil when not set
}

// AvailabilityDay holds the available time slots for a single calendar day.
type AvailabilityDay struct {
	Date  string   `json:"date"`  // YYYY-MM-DD
	Slots []string `json:"slots"` // ["HH:MM", ...]
}

// AvailabilityResponse mirrors GET /bot/availability response.
type AvailabilityResponse struct {
	DoctorID               int64             `json:"doctor_id"`
	ServiceID              int64             `json:"service_id"`
	ServiceDurationMinutes int               `json:"service_duration_minutes"`
	Availability           []AvailabilityDay `json:"availability"`
}

// CreateAppointmentInput is the body sent to POST /bot/appointments.
// Field names match the backend createAppointmentRequest struct.
type CreateAppointmentInput struct {
	PatientTelegramID       *int64  `json:"patient_telegram_id,omitempty"`
	PatientTelegramUsername *string `json:"patient_telegram_username,omitempty"`
	PatientName             string  `json:"patient_name"`
	PatientPhone            string  `json:"patient_phone"`
	DoctorID                int64   `json:"doctor_id"`
	ServiceID               int64   `json:"service_id"`
	StartAt                 string  `json:"start_at"` // RFC3339
	PatientComment          *string `json:"patient_comment,omitempty"`
}

// AppointmentResult is the minimal confirmation data the bot needs after booking.
type AppointmentResult struct {
	ID      int64  `json:"id"`
	StartAt string `json:"start_at"` // RFC3339
	EndAt   string `json:"end_at"`   // RFC3339
}
