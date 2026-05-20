package apperrors

import "errors"

var (
	// Generic
	ErrNotFound = errors.New("not found")

	// Auth
	ErrUnauthorized = errors.New("unauthorized")
	ErrInactiveUser = errors.New("user is inactive")

	// CRUD
	ErrConflict = errors.New("conflict")

	// Scheduling (CLAUDE.md §3.2)
	ErrSlotTaken      = errors.New("time slot already taken")
	ErrOutsideHours   = errors.New("outside working hours")
	ErrDoctorInactive = errors.New("doctor is inactive")
)
