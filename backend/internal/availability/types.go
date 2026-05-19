package availability

import "time"

// Slot represents a bookable time interval.
type Slot struct {
	Start time.Time
	End   time.Time
}

// WorkingInterval is a resolved, date-specific working window.
type WorkingInterval struct {
	Start time.Time
	End   time.Time
}

// RegularSchedule is one entry of the doctor's weekly template.
// Start and End carry time-of-day only; the date part is ignored.
type RegularSchedule struct {
	DayOfWeek time.Weekday
	Start     time.Time
	End       time.Time
}

// Exception overrides the regular schedule for a specific date.
// Start and End are nil when Type == "day_off".
type Exception struct {
	Date  time.Time
	Type  string // "day_off" | "custom_working_hours"
	Start *time.Time
	End   *time.Time
}

// CalculatorInput is everything the pure calculator needs.
type CalculatorInput struct {
	Date                 time.Time
	ServiceDuration      time.Duration
	RegularSchedule      []RegularSchedule
	Exceptions           []Exception
	ExistingAppointments []Slot
	SlotStep             time.Duration
}

// DayAvailability is the output for one calendar day.
type DayAvailability struct {
	Date  time.Time
	Slots []Slot
}
