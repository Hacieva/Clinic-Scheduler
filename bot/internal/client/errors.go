package client

import "errors"

var (
	// ErrSlotTaken is returned when the backend responds 409 on appointment creation.
	// The flow layer resets the session and shows a user-friendly message.
	ErrSlotTaken = errors.New("time slot already taken")

	// ErrUnauthorized indicates a misconfigured BOT_API_SECRET (401/403 from backend).
	// Must NOT be shown to the user — log and return a generic error.
	ErrUnauthorized = errors.New("bot authentication rejected by backend")

	// ErrNotFound is returned on 404 responses.
	ErrNotFound = errors.New("resource not found")

	// ErrTemporary covers network errors, context timeouts, and 5xx responses.
	// Safe to present to the user as "something went wrong, try again later".
	ErrTemporary = errors.New("backend temporarily unavailable")
)
