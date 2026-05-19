package availability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// refDate is 2026-05-20 (Wednesday). Used as the canonical test day.
var refDate = time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)

// tod builds a time-of-day value (date part zeroed).
func tod(h, m int) time.Time {
	return time.Date(0, 1, 1, h, m, 0, 0, time.UTC)
}

// dt builds a timestamp on refDate with the given hour and minute.
func dt(h, m int) time.Time {
	return time.Date(2026, 5, 20, h, m, 0, 0, time.UTC)
}

// baseInput returns a CalculatorInput for refDate (Wednesday) with
// working hours 10:00–12:00, 60-min service, 30-min step, no bookings.
func baseInput() CalculatorInput {
	return CalculatorInput{
		Date:            refDate,
		ServiceDuration: 60 * time.Minute,
		SlotStep:        30 * time.Minute,
		RegularSchedule: []RegularSchedule{
			{DayOfWeek: time.Wednesday, Start: tod(10, 0), End: tod(12, 0)},
		},
	}
}

// TestCalculateSlots_BasicDay: no bookings, 10:00–12:00, 60-min service, 30-min step.
// Expected: [10:00–11:00, 10:30–11:30, 11:00–12:00]
func TestCalculateSlots_BasicDay(t *testing.T) {
	result := Calculate(baseInput())

	require.Len(t, result, 3)
	assert.Equal(t, dt(10, 0), result[0].Start)
	assert.Equal(t, dt(11, 0), result[0].End)
	assert.Equal(t, dt(10, 30), result[1].Start)
	assert.Equal(t, dt(11, 30), result[1].End)
	assert.Equal(t, dt(11, 0), result[2].Start)
	assert.Equal(t, dt(12, 0), result[2].End)
}

// TestCalculateSlots_WithBookedSlot: 10:00–11:00 is booked.
// 10:00–11:00 and 10:30–11:30 both conflict → only 11:00–12:00 remains.
func TestCalculateSlots_WithBookedSlot(t *testing.T) {
	input := baseInput()
	input.ExistingAppointments = []Slot{
		{Start: dt(10, 0), End: dt(11, 0)},
	}

	result := Calculate(input)

	require.Len(t, result, 1)
	assert.Equal(t, dt(11, 0), result[0].Start)
	assert.Equal(t, dt(12, 0), result[0].End)
}

// TestCalculateSlots_DayOff: exception type=day_off overrides regular schedule.
func TestCalculateSlots_DayOff(t *testing.T) {
	input := baseInput()
	input.Exceptions = []Exception{
		{Date: refDate, Type: "day_off"},
	}

	result := Calculate(input)

	assert.Empty(t, result)
}

// TestCalculateSlots_CustomWorkingHours: exception 12:00–15:00 replaces 10:00–12:00.
// Expected slots (60-min, 30-min step): 12:00, 12:30, 13:00, 13:30, 14:00 → 5 slots.
func TestCalculateSlots_CustomWorkingHours(t *testing.T) {
	input := baseInput()
	start, end := tod(12, 0), tod(15, 0)
	input.Exceptions = []Exception{
		{Date: refDate, Type: "custom_working_hours", Start: &start, End: &end},
	}

	result := Calculate(input)

	require.Len(t, result, 5)
	assert.Equal(t, dt(12, 0), result[0].Start)
	assert.Equal(t, dt(13, 0), result[0].End)
	assert.Equal(t, dt(14, 0), result[4].Start)
	assert.Equal(t, dt(15, 0), result[4].End)
}

// TestCalculateSlots_NonWorkingDay: Wednesday not in schedule (only Monday listed).
func TestCalculateSlots_NonWorkingDay(t *testing.T) {
	input := CalculatorInput{
		Date:            refDate, // Wednesday
		ServiceDuration: 60 * time.Minute,
		SlotStep:        30 * time.Minute,
		RegularSchedule: []RegularSchedule{
			{DayOfWeek: time.Monday, Start: tod(10, 0), End: tod(18, 0)},
		},
	}

	result := Calculate(input)

	assert.Empty(t, result)
}

// TestCalculateSlots_TwoIntervals: 10:00–13:00 and 14:00–17:00 (lunch break 13–14).
// Service 60-min, step 30-min.
// Interval 1 (10–13): 10:00, 10:30, 11:00, 11:30, 12:00 → 5 slots
// Interval 2 (14–17): 14:00, 14:30, 15:00, 15:30, 16:00 → 5 slots
// No slot spans the lunch gap.
func TestCalculateSlots_TwoIntervals(t *testing.T) {
	input := CalculatorInput{
		Date:            refDate,
		ServiceDuration: 60 * time.Minute,
		SlotStep:        30 * time.Minute,
		RegularSchedule: []RegularSchedule{
			{DayOfWeek: time.Wednesday, Start: tod(10, 0), End: tod(13, 0)},
			{DayOfWeek: time.Wednesday, Start: tod(14, 0), End: tod(17, 0)},
		},
	}

	result := Calculate(input)

	require.Len(t, result, 10)
	// Last slot of morning block
	assert.Equal(t, dt(12, 0), result[4].Start)
	assert.Equal(t, dt(13, 0), result[4].End)
	// First slot of afternoon block
	assert.Equal(t, dt(14, 0), result[5].Start)
	assert.Equal(t, dt(15, 0), result[5].End)
}

// TestCalculateSlots_ServiceTooLong: 90-min service in a 60-min window → no slots.
func TestCalculateSlots_ServiceTooLong(t *testing.T) {
	input := CalculatorInput{
		Date:            refDate,
		ServiceDuration: 90 * time.Minute,
		SlotStep:        30 * time.Minute,
		RegularSchedule: []RegularSchedule{
			{DayOfWeek: time.Wednesday, Start: tod(10, 0), End: tod(11, 0)},
		},
	}

	result := Calculate(input)

	assert.Empty(t, result)
}

// TestCalculateSlots_BoundaryConditions: booked slot ends exactly at window start (9:30–10:00).
// Must NOT block [10:00–11:00] — boundary is exclusive.
func TestCalculateSlots_BoundaryConditions(t *testing.T) {
	input := baseInput()
	input.ExistingAppointments = []Slot{
		{Start: dt(9, 30), End: dt(10, 0)},
	}

	result := Calculate(input)

	require.Len(t, result, 3)
	assert.Equal(t, dt(10, 0), result[0].Start)
	assert.Equal(t, dt(10, 30), result[1].Start)
	assert.Equal(t, dt(11, 0), result[2].Start)
}

// TestCalculateSlots_BackToBackBookings: 10:00–10:30 and 10:30–11:00 both booked.
// Only 11:00–12:00 is free.
func TestCalculateSlots_BackToBackBookings(t *testing.T) {
	input := baseInput()
	input.ExistingAppointments = []Slot{
		{Start: dt(10, 0), End: dt(10, 30)},
		{Start: dt(10, 30), End: dt(11, 0)},
	}

	result := Calculate(input)

	require.Len(t, result, 1)
	assert.Equal(t, dt(11, 0), result[0].Start)
	assert.Equal(t, dt(12, 0), result[0].End)
}

// TestCalculateSlots_ZeroStep: SlotStep=0 must return empty, not hang.
func TestCalculateSlots_ZeroStep(t *testing.T) {
	input := baseInput()
	input.SlotStep = 0

	result := Calculate(input)

	assert.Empty(t, result)
}

// TestCalculateSlots_InvertedInterval: schedule End before Start → no slots.
func TestCalculateSlots_InvertedInterval(t *testing.T) {
	input := CalculatorInput{
		Date:            refDate,
		ServiceDuration: 60 * time.Minute,
		SlotStep:        30 * time.Minute,
		RegularSchedule: []RegularSchedule{
			{DayOfWeek: time.Wednesday, Start: tod(14, 0), End: tod(10, 0)},
		},
	}

	result := Calculate(input)

	assert.Empty(t, result)
}

// TestResolveIntervals_CustomHoursNilEnd: custom_working_hours with nil End falls through to nil.
func TestResolveIntervals_CustomHoursNilEnd(t *testing.T) {
	start := tod(12, 0)
	input := baseInput()
	input.Exceptions = []Exception{
		{Date: refDate, Type: "custom_working_hours", Start: &start, End: nil},
	}

	result := Calculate(input)

	assert.Empty(t, result)
}

// TestSameDay_MixedTimezones: appointment stored as UTC must match a local-timezone day.
// 2026-05-19 22:00 UTC = 2026-05-20 01:00 UTC+3 — same local calendar day.
func TestSameDay_MixedTimezones(t *testing.T) {
	loc := time.FixedZone("UTC+3", 3*3600)
	dayLocal := time.Date(2026, 5, 20, 0, 0, 0, 0, loc)
	apptUTC := time.Date(2026, 5, 19, 22, 0, 0, 0, time.UTC) // 01:00 May 20 in UTC+3

	assert.True(t, sameDay(dayLocal, apptUTC))
	assert.False(t, sameDay(dayLocal, time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)))
}
