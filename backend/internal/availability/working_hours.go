package availability

import "time"

// IsWithinWorkingHours reports whether [startAt, endAt) falls entirely within
// one of the resolved working intervals for startAt's calendar day.
// input.Date must be set to midnight of startAt's day (same location).
func IsWithinWorkingHours(startAt, endAt time.Time, input CalculatorInput) bool {
	intervals := resolveIntervals(input)
	for _, iv := range intervals {
		if !startAt.Before(iv.Start) && !endAt.After(iv.End) {
			return true
		}
	}
	return false
}
