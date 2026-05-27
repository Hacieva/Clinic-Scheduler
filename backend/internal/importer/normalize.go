package importer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	// rexLeadingCode strips service codes like "ГИН009 " from the start of a name.
	// \p{Lu} matches any Unicode uppercase letter (handles Cyrillic codes).
	rexLeadingCode = regexp.MustCompile(`^\p{Lu}{2,5}\d{2,4}\s+`)

	// rexInterval matches HH:MM-HH:MM (hour may be 1 or 2 digits).
	rexInterval = regexp.MustCompile(`^(\d{1,2}):(\d{2})-(\d{1,2}):(\d{2})$`)

	// rexParenthetical strips audience qualifiers like "(детский)", "(взрослый)".
	rexParenthetical = regexp.MustCompile(`\s*\([^)]*\)`)
)

// abbreviations expands Russian shortforms before punctuation is stripped.
// Applied after lowercase so matching is case-insensitive by construction.
var abbreviations = [][2]string{
	{"к-ция", "консультация"},
	{"конс.", "консультация"},
	{"перв.", "первичная"},
	{"повт.", "повторная"},
	{"дет.", "детский"},
	{"взр.", "взрослый"},
}

// NormalizeServiceName normalizes a service name for fuzzy matching.
// The result is NOT stored in the DB — it is only used for name matching.
func NormalizeServiceName(s string) string {
	s = strings.TrimSpace(s)
	// 1. Strip leading service code before lowercasing (codes are uppercase).
	s = rexLeadingCode.ReplaceAllString(s, "")
	// 2. Lowercase.
	s = strings.ToLower(s)
	// 3. Expand abbreviations while dots are still present.
	for _, ab := range abbreviations {
		s = strings.ReplaceAll(s, ab[0], ab[1])
	}
	// 4. Strip non-letter, non-digit characters.
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			return r
		}
		return ' '
	}, s)
	// 5. Collapse whitespace.
	return strings.Join(strings.Fields(s), " ")
}

// ParseWorkingHours parses a freeform schedule string from the spreadsheet.
//
// dayOfWeek: 1=Monday … 6=Saturday; 7=Sunday.
//
// Returns zero rows (not an error) for "" or "вых" — doctor is off that day.
// Returns an error for strings that do not match any recognized pattern.
func ParseWorkingHours(s string, dayOfWeek int) ([]WorkingHoursRow, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "вых") {
		return nil, nil
	}
	segments := strings.Split(s, "/")
	result := make([]WorkingHoursRow, 0, len(segments))
	for _, seg := range segments {
		row, err := parseInterval(strings.TrimSpace(seg), dayOfWeek)
		if err != nil {
			return nil, fmt.Errorf("unparseable schedule %q: %w", s, err)
		}
		result = append(result, row)
	}
	return result, nil
}

func parseInterval(s string, dayOfWeek int) (WorkingHoursRow, error) {
	m := rexInterval.FindStringSubmatch(s)
	if m == nil {
		return WorkingHoursRow{}, fmt.Errorf("does not match HH:MM-HH:MM")
	}
	sh, _ := strconv.Atoi(m[1])
	sm, _ := strconv.Atoi(m[2])
	eh, _ := strconv.Atoi(m[3])
	em, _ := strconv.Atoi(m[4])
	start := time.Date(2000, 1, 1, sh, sm, 0, 0, time.UTC)
	end := time.Date(2000, 1, 1, eh, em, 0, 0, time.UTC)
	if !start.Before(end) {
		return WorkingHoursRow{}, fmt.Errorf("start %02d:%02d >= end %02d:%02d", sh, sm, eh, em)
	}
	return WorkingHoursRow{DayOfWeek: dayOfWeek, StartTime: start, EndTime: end}, nil
}

// MapBookingType maps a raw booking_type value from doctors_v2 to normalized enums.
// Returns unresolved=true when the raw value was not recognized (defaults applied).
func MapBookingType(raw string) (kind DoctorKind, mode BookingMode, unresolved bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "запись_и_живая", "по_записи_и_живой", "запись и живая", "mixed":
		return DoctorKindStaff, BookingModeMixed, false
	case "по_записи", "по записи", "запись", "appointment":
		return DoctorKindStaff, BookingModeAppointmentOnly, false
	case "по_живой", "по живой", "живая", "живой", "queue":
		return DoctorKindStaff, BookingModeQueueOnly, false
	case "приезжающий", "приезжающая", "приезжающие", "visiting":
		return DoctorKindVisiting, BookingModeAppointmentOnly, false
	default:
		return DoctorKindStaff, BookingModeAppointmentOnly, true
	}
}

// MapAudience maps a raw audience string to PatientAudience.
// Returns nil if the value is absent — no doctor-level restriction is recorded.
func MapAudience(raw string) *PatientAudience {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "детский", "дети", "child":
		a := AudienceChild
		return &a
	case "взрослый", "взрослые", "adult":
		a := AudienceAdult
		return &a
	case "оба", "оба пола", "детский/взрослый", "взрослый/детский",
		"дети/взрослые", "оба/взрослые", "both":
		a := AudienceBoth
		return &a
	default:
		return nil
	}
}

// MapPatientType maps a raw patient_type value from doctor_services_v2.
// Returns "both" when absent or unrecognized.
func MapPatientType(raw string) PatientAudience {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "дети", "детский", "child":
		return AudienceChild
	case "взрослые", "взрослый", "adult":
		return AudienceAdult
	default:
		return AudienceBoth
	}
}

// ParsePriceKopecks converts a price string like "7000.00" into kopecks (700000).
// Returns (0, false) if the string is empty or cannot be parsed.
func ParsePriceKopecks(s string) (int64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, " ", ""))
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0, false
	}
	return int64(f * 100), true
}

// ParseDuration converts a duration string like "30" or "от 15" into minutes.
// Returns (0, false) if the string is empty or cannot be parsed.
func ParseDuration(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// Strip qualifier "от " (e.g. "от 15 минут" → "15").
	s = strings.TrimPrefix(s, "от ")
	// Take only the leading integer part (e.g. "30 минут" → "30").
	if i := strings.IndexByte(s, ' '); i > 0 {
		s = s[:i]
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// SplitName splits a full name "Фамилия Имя Отчество" into components.
func SplitName(full string) (lastName, firstName, middleName string) {
	parts := strings.Fields(strings.TrimSpace(full))
	switch len(parts) {
	case 0:
		return "", "", ""
	case 1:
		return parts[0], "", ""
	case 2:
		return parts[0], parts[1], ""
	default:
		return parts[0], parts[1], strings.Join(parts[2:], " ")
	}
}

// ParseDirections splits a specialty string like "Педиатр, пульмонолог" into
// normalized direction names, stripping parenthetical audience qualifiers.
func ParseDirections(specialty string) []string {
	if specialty == "" {
		return nil
	}
	parts := strings.Split(specialty, ",")
	seen := make(map[string]bool, len(parts))
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		// Remove "(детский)", "(взрослый)" etc.
		p = rexParenthetical.ReplaceAllString(p, "")
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Capitalize first letter.
		runes := []rune(p)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			p = string(runes)
		}
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}
