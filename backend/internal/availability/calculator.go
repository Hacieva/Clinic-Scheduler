package availability

import (
	"sort"
	"time"
)

// Calculate returns all available slots for a single day.
// It is a pure function: no I/O, no side effects.
func Calculate(input CalculatorInput) []Slot {
	intervals := resolveIntervals(input)
	if len(intervals) == 0 {
		return nil
	}

	var result []Slot
	for _, iv := range intervals {
		result = append(result, slotsForInterval(iv, input)...)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Start.Before(result[j].Start)
	})

	return result
}

// resolveIntervals determines which working windows apply on input.Date.
// Exception entries take priority over the regular weekly schedule.
func resolveIntervals(input CalculatorInput) []WorkingInterval {
	for _, ex := range input.Exceptions {
		if sameDay(ex.Date, input.Date) {
			if ex.Type == "day_off" {
				return nil
			}
			if ex.Type == "custom_working_hours" && ex.Start != nil && ex.End != nil {
				return []WorkingInterval{{
					Start: applyTimeToDate(input.Date, *ex.Start),
					End:   applyTimeToDate(input.Date, *ex.End),
				}}
			}
			return nil
		}
	}

	weekday := input.Date.Weekday()
	var intervals []WorkingInterval
	for _, s := range input.RegularSchedule {
		if s.DayOfWeek == weekday {
			intervals = append(intervals, WorkingInterval{
				Start: applyTimeToDate(input.Date, s.Start),
				End:   applyTimeToDate(input.Date, s.End),
			})
		}
	}
	return intervals
}

// slotsForInterval generates all non-conflicting slots within one working window.
func slotsForInterval(iv WorkingInterval, input CalculatorInput) []Slot {
	if input.SlotStep <= 0 {
		return nil
	}
	var slots []Slot
	for start := iv.Start; !start.Add(input.ServiceDuration).After(iv.End); start = start.Add(input.SlotStep) {
		end := start.Add(input.ServiceDuration)
		if !overlapsAny(start, end, input.ExistingAppointments) {
			slots = append(slots, Slot{Start: start, End: end})
		}
	}
	return slots
}

// overlapsAny reports whether [start, end) intersects any booked slot.
// Two half-open intervals [a,b) and [c,d) overlap iff a < d AND b > c.
func overlapsAny(start, end time.Time, booked []Slot) bool {
	for _, b := range booked {
		if start.Before(b.End) && end.After(b.Start) {
			return true
		}
	}
	return false
}

// sameDay reports whether a and b fall on the same calendar date in a's timezone.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()
	return ay == by && am == bm && ad == bd
}

// applyTimeToDate combines the calendar date from date with the clock time from tod.
func applyTimeToDate(date, tod time.Time) time.Time {
	return time.Date(
		date.Year(), date.Month(), date.Day(),
		tod.Hour(), tod.Minute(), tod.Second(), 0,
		date.Location(),
	)
}
