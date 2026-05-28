package model

import (
	"encoding/json"
	"time"
)

// Patient mirrors the patients table.
// Patients have no user account; identified by TelegramUserID and Phone.
type Patient struct {
	ID                int64      `json:"id"`
	TelegramUserID    *int64     `json:"telegram_user_id,omitempty"`
	TelegramUsername  *string    `json:"telegram_username,omitempty"`
	FullName          string     `json:"full_name"`
	Phone             string     `json:"phone"`
	DateOfBirth       *time.Time `json:"date_of_birth,omitempty"`
	Email             *string    `json:"email,omitempty"`
	Comment           *string    `json:"comment,omitempty"`
	Source            string     `json:"source"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastAppointmentAt *time.Time `json:"last_appointment_at,omitempty"`
}

// BotSession mirrors the bot_sessions table.
// State and Data persist FSM state between bot restarts.
type BotSession struct {
	ID             int64           `json:"id"`
	TelegramUserID int64           `json:"telegram_user_id"`
	State          string          `json:"state"`
	Data           json.RawMessage `json:"data"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
