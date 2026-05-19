package model

import (
	"encoding/json"
	"time"
)

// Patient mirrors the patients table.
// Patients have no user account; identified by TelegramUserID and Phone.
type Patient struct {
	ID               int64     `json:"id"`
	TelegramUserID   *int64    `json:"telegram_user_id"`
	TelegramUsername *string   `json:"telegram_username"`
	FullName         string    `json:"full_name"`
	Phone            string    `json:"phone"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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
