package model

import "time"

// Service mirrors the services table.
// DoctorID is nullable — services now live in a global catalog;
// the field is kept temporarily for bot backward compatibility (TODO: remove after bot migrates to doctor_services).
// Price is stored in kopecks (e.g. 300050 = 3000.50 ₽).
type Service struct {
	ID              int64     `json:"id"`
	DoctorID        *int64    `json:"doctor_id"` // TODO: legacy; nil for global-catalog services
	DirectionID     *int64    `json:"direction_id"` // nullable — directions optional for catalog services
	Category        *string   `json:"category"`
	Name            string    `json:"name"`
	Code            *string   `json:"code,omitempty"` // Medlock service code, e.g. АК001
	Description     *string   `json:"description"`
	DurationMinutes int       `json:"duration_minutes"`
	Price           *int64    `json:"price"` // kopecks (e.g. 300050 = 3000.50 ₽)
	IsActive        bool      `json:"is_active"`
	PatientType     *string   `json:"patient_type,omitempty"` // populated from doctor_services junction
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
