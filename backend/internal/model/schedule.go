package model

import "time"

// ExceptionType mirrors CHECK (type IN ('day_off', 'custom_working_hours'))
// on the doctor_schedule_exceptions table.
type ExceptionType string

const (
	ExceptionTypeDayOff             ExceptionType = "day_off"
	ExceptionTypeCustomWorkingHours ExceptionType = "custom_working_hours"
)

// WorkingHours mirrors the doctor_working_hours table.
// StartTime and EndTime hold TIME values (date part is zero).
// DayOfWeek is 1–7 per CHECK (day_of_week BETWEEN 1 AND 7).
type WorkingHours struct {
	ID        int64     `json:"id"`
	DoctorID  int64     `json:"doctor_id"`
	DayOfWeek int       `json:"day_of_week"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScheduleException mirrors the doctor_schedule_exceptions table.
// Date holds a DATE value (time part is zero).
// StartTime and EndTime are nullable: nil when Type == ExceptionTypeDayOff.
type ScheduleException struct {
	ID        int64         `json:"id"`
	DoctorID  int64         `json:"doctor_id"`
	Date      time.Time     `json:"date"`
	Type      ExceptionType `json:"type"`
	StartTime *time.Time    `json:"start_time"`
	EndTime   *time.Time    `json:"end_time"`
	Comment   *string       `json:"comment"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
