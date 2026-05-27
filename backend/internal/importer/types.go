package importer

import "time"

// DoctorKind describes the employment/presence relationship with the clinic.
type DoctorKind string

// BookingMode describes how patients access a doctor.
type BookingMode string

// PatientAudience describes the patient age group served.
type PatientAudience string

const (
	DoctorKindStaff    DoctorKind = "staff"
	DoctorKindVisiting DoctorKind = "visiting"

	BookingModeAppointmentOnly BookingMode = "appointment_only"
	BookingModeQueueOnly       BookingMode = "queue_only"
	BookingModeMixed           BookingMode = "mixed"

	AudienceAdult PatientAudience = "adult"
	AudienceChild PatientAudience = "child"
	AudienceBoth  PatientAudience = "both"
)

// BranchRow is a clinic branch ready for import.
type BranchRow struct {
	Name    string
	Address string
}

// DirectionRow is a specialization/direction ready for import.
type DirectionRow struct {
	Name string // canonical display name, e.g. "Гинекология"
}

// WorkingHoursRow is one structured working interval.
// The date part of StartTime/EndTime is always 2000-01-01 (sentinel).
type WorkingHoursRow struct {
	DayOfWeek int       // 1=Mon … 6=Sat; 7=Sun
	StartTime time.Time // date part ignored; only time matters
	EndTime   time.Time
}

// DoctorRow is a doctor ready for import.
type DoctorRow struct {
	SourceID     string           // D001…D073, V001…VNN for visiting
	FullName     string           // original full name from source
	FirstName    string
	LastName     string
	MiddleName   string           // empty if not provided
	BranchName   string           // matches BranchRow.Name; empty for visiting
	Directions   []string         // normalized direction names
	Audience     *PatientAudience // nil = no doctor-level restriction
	DoctorKind   DoctorKind
	BookingMode  BookingMode
	IsActive     bool
	WorkingHours []WorkingHoursRow
}

// ServiceRow is a catalog service ready for import.
type ServiceRow struct {
	Code            string // Medlock code, e.g. "ГИН009"; empty if absent
	Name            string // canonical Medlock name
	Category        string // derived from section heading in price sheet
	DurationMinutes int    // 0 if not specified
	Price           int64  // kopecks (price × 100); 0 if not specified
	BranchName      string // branch whose price list this came from
}

// AssignmentRow is a doctor–service assignment ready for import.
type AssignmentRow struct {
	DoctorSourceID  string          // matches DoctorRow.SourceID
	DoctorName      string          // original name, for traceability
	ServiceName     string          // original name from doctor_services_v2
	ServiceCode     string          // matched canonical code; empty if unmatched
	PatientType     PatientAudience
	Price           int64  // kopecks; may differ from ServiceRow.Price
	MatchConfidence string // "exact", "fuzzy", "override", "unmatched"
}

// ImportWarning records a non-fatal issue found during parsing.
type ImportWarning struct {
	Entity   string // "doctor", "service", "schedule", "assignment"
	SourceID string // e.g. "D042" or raw name
	Kind     string // "unresolved_schedule", "unmatched_service", etc.
	Detail   string
}

// ImportPlan is the full parsed result ready for dry-run summary or execution.
type ImportPlan struct {
	Branches    []BranchRow
	Directions  []DirectionRow
	Doctors     []DoctorRow
	Services    []ServiceRow
	Assignments []AssignmentRow
	Warnings    []ImportWarning
}

func (p *ImportPlan) warn(entity, sourceID, kind, detail string) {
	p.Warnings = append(p.Warnings, ImportWarning{
		Entity:   entity,
		SourceID: sourceID,
		Kind:     kind,
		Detail:   detail,
	})
}
