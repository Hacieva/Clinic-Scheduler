package model

import "time"

// Doctor mirrors the doctors table.
type Doctor struct {
	ID          int64     `json:"id"`
	UserID      *int64    `json:"user_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	MiddleName  *string   `json:"middle_name"`
	Cabinet     *string   `json:"cabinet"`
	BranchID    *int64    `json:"branch_id"`
	Phone       *string   `json:"phone"`
	Description *string   `json:"description"`
	PhotoURL    *string   `json:"photo_url"`
	DoctorKind  string    `json:"doctor_kind"` // "staff" or "visiting"
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DoctorDirection mirrors the doctor_directions table (M2M junction).
type DoctorDirection struct {
	ID          int64     `json:"id"`
	DoctorID    int64     `json:"doctor_id"`
	DirectionID int64     `json:"direction_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// DoctorWithDirections is the read model returned by queries that JOIN
// doctors with their assigned directions.
type DoctorWithDirections struct {
	Doctor
	Directions []Direction `json:"directions"`
}
