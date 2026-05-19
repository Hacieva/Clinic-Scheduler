package model

import (
	"encoding/json"
	"time"
)

// UserRole mirrors CHECK (role IN ('admin', 'doctor')) on the users table.
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleDoctor UserRole = "doctor"
)

// User mirrors the users table.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuditLog mirrors the audit_logs table.
type AuditLog struct {
	ID         int64           `json:"id"`
	UserID     int64           `json:"user_id"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   *int64          `json:"entity_id"`
	OldValues  json.RawMessage `json:"old_values"`
	NewValues  json.RawMessage `json:"new_values"`
	IPAddress  *string         `json:"ip_address"`
	UserAgent  *string         `json:"user_agent"`
	CreatedAt  time.Time       `json:"created_at"`
}
