package model

import "time"

// Service mirrors the services table.
// Price is nullable (DECIMAL(10,2)); stored as *float64 for simplicity in MVP.
type Service struct {
	ID              int64     `json:"id"`
	DoctorID        int64     `json:"doctor_id"`
	DirectionID     int64     `json:"direction_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	DurationMinutes int    `json:"duration_minutes"`
	Price           *int64 `json:"price"` // kopecks (e.g. 300050 = 3000.50 ₽)
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
