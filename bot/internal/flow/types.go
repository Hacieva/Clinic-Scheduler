package flow

// FSM state constants for the Telegram booking flow.
// Transition order:
//
//	Start → ChooseDirection → ChooseDoctor → ChooseService →
//	ChooseDate → ChooseTime → EnterName → EnterPhone → Confirm
const (
	StateStart           = "start"
	StateChooseDirection = "choose_direction"
	StateChooseDoctor    = "choose_doctor"
	StateChooseService   = "choose_service"
	StateChooseDate      = "choose_date"
	StateChooseTime      = "choose_time"
	StateEnterName       = "enter_name"
	StateEnterPhone      = "enter_phone"
	StateConfirm         = "confirm"
)

// Callback data constants for inline keyboard buttons.
//
// Prefixed callbacks use format "<prefix>:<value>", e.g. "direction:1", "time:10:00".
// Action callbacks are bare strings: "confirm", "cancel".
//
// Unknown callback data for the current FSM state must be treated as a no-op:
// session state must NOT be mutated.
const (
	CallbackPrefixDirection = "direction"
	CallbackPrefixDoctor    = "doctor"
	CallbackPrefixService   = "service"
	CallbackPrefixDate      = "date"
	CallbackPrefixTime      = "time"
	CallbackConfirm         = "confirm"
	// CallbackCancel is handled globally before FSM dispatch.
	// It deletes the session and is idempotent (safe when no session exists).
	CallbackCancel = "cancel"
)
