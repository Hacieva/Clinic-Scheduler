package importer

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExecuteResult summarises what was written to the database.
type ExecuteResult struct {
	Branches        int
	Directions      int
	Services        int
	Doctors         int
	WorkingHourRows int
	DoctorDirs      int
	DoctorServices  int
	Skipped         int // unmatched assignments
}

// Execute writes plan to the database inside a single transaction.
// It is idempotent: running it twice produces the same final state.
// Patients are never imported here.
func Execute(ctx context.Context, db *pgxpool.Pool, plan *ImportPlan) (*ExecuteResult, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	res := &ExecuteResult{}

	// ── 1. Branches ─────────────────────────────────────────────────────────
	branchIDs := make(map[string]int64, len(plan.Branches))
	for _, b := range plan.Branches {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO branches (name, address)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE
				SET address    = EXCLUDED.address,
				    updated_at = NOW()
			RETURNING id`,
			b.Name, nullableString(b.Address),
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert branch %q: %w", b.Name, err)
		}
		branchIDs[b.Name] = id
		res.Branches++
	}

	// ── 2. Directions ────────────────────────────────────────────────────────
	dirIDs := make(map[string]int64, len(plan.Directions))
	for _, d := range plan.Directions {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO directions (name)
			VALUES ($1)
			ON CONFLICT (name) DO UPDATE
				SET updated_at = NOW()
			RETURNING id`,
			d.Name,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert direction %q: %w", d.Name, err)
		}
		dirIDs[d.Name] = id
		res.Directions++
	}

	// ── 3. Services ──────────────────────────────────────────────────────────
	// Build code→id map for assignment lookup.
	serviceCodeIDs := make(map[string]int64, len(plan.Services))
	for _, s := range plan.Services {
		dur := s.DurationMinutes
		if dur == 0 {
			dur = 30 // minimum slot per business rules; clinic corrects via UI
		}

		var id int64
		if s.Code != "" {
			// Coded service: upsert on code.
			err := tx.QueryRow(ctx, `
				INSERT INTO services (code, name, category, duration_minutes, price, is_active)
				VALUES ($1, $2, $3, $4, $5, true)
				ON CONFLICT (code) DO UPDATE
					SET name             = EXCLUDED.name,
					    category         = EXCLUDED.category,
					    duration_minutes = EXCLUDED.duration_minutes,
					    price            = EXCLUDED.price,
					    updated_at       = NOW()
				RETURNING id`,
				s.Code, s.Name, s.Category, dur, s.Price,
			).Scan(&id)
			if err != nil {
				return nil, fmt.Errorf("upsert service code=%q name=%q: %w", s.Code, s.Name, err)
			}
			serviceCodeIDs[s.Code] = id
		} else {
			// Uncoded service: find by name or insert.
			err := tx.QueryRow(ctx, `
				SELECT id FROM services WHERE LOWER(name) = LOWER($1) LIMIT 1`,
				s.Name,
			).Scan(&id)
			if err == pgx.ErrNoRows {
				err = tx.QueryRow(ctx, `
					INSERT INTO services (name, category, duration_minutes, price, is_active)
					VALUES ($1, $2, $3, $4, true)
					RETURNING id`,
					s.Name, s.Category, dur, s.Price,
				).Scan(&id)
				if err != nil {
					return nil, fmt.Errorf("insert uncoded service %q: %w", s.Name, err)
				}
			} else if err != nil {
				return nil, fmt.Errorf("lookup uncoded service %q: %w", s.Name, err)
			}
		}
		res.Services++
		_ = id // uncoded services are referenced only by assignments below via code lookup
	}

	// ── 4. Doctors ───────────────────────────────────────────────────────────
	doctorIDs := make(map[string]int64, len(plan.Doctors)) // sourceID → db id
	for _, d := range plan.Doctors {
		var branchID *int64
		if d.BranchName != "" {
			if bid, ok := branchIDs[d.BranchName]; ok {
				branchID = &bid
			}
		}

		var audience *string
		if d.Audience != nil {
			s := string(*d.Audience)
			audience = &s
		}

		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO doctors
				(import_source_id, first_name, last_name, middle_name,
				 branch_id, doctor_kind, booking_mode, audience, is_active)
			VALUES
				($1, $2, $3, $4,
				 $5, $6::doctor_kind, $7::booking_mode, $8::patient_audience, $9)
			ON CONFLICT (import_source_id) DO UPDATE
				SET first_name   = EXCLUDED.first_name,
				    last_name    = EXCLUDED.last_name,
				    middle_name  = EXCLUDED.middle_name,
				    branch_id    = EXCLUDED.branch_id,
				    doctor_kind  = EXCLUDED.doctor_kind,
				    booking_mode = EXCLUDED.booking_mode,
				    audience     = EXCLUDED.audience,
				    updated_at   = NOW()
			RETURNING id`,
			d.SourceID,
			d.FirstName, d.LastName, nullableString(d.MiddleName),
			branchID,
			string(d.DoctorKind), string(d.BookingMode), audience,
			d.IsActive,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert doctor %q (sourceID=%s): %w", d.FullName, d.SourceID, err)
		}
		doctorIDs[d.SourceID] = id
		res.Doctors++

		// ── 4a. Working hours: replace on every run ──────────────────────────
		if _, err := tx.Exec(ctx, `DELETE FROM doctor_working_hours WHERE doctor_id = $1`, id); err != nil {
			return nil, fmt.Errorf("delete working_hours doctor=%s: %w", d.SourceID, err)
		}
		for _, wh := range d.WorkingHours {
			if _, err := tx.Exec(ctx, `
				INSERT INTO doctor_working_hours (doctor_id, day_of_week, start_time, end_time)
				VALUES ($1, $2, $3, $4)`,
				id, wh.DayOfWeek, wh.StartTime, wh.EndTime,
			); err != nil {
				return nil, fmt.Errorf("insert working_hours doctor=%s dow=%d: %w", d.SourceID, wh.DayOfWeek, err)
			}
			res.WorkingHourRows++
		}

		// ── 4b. Doctor–direction links ───────────────────────────────────────
		for _, dirName := range d.Directions {
			dirID, ok := dirIDs[dirName]
			if !ok {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO doctor_directions (doctor_id, direction_id)
				VALUES ($1, $2)
				ON CONFLICT (doctor_id, direction_id) DO NOTHING`,
				id, dirID,
			); err != nil {
				return nil, fmt.Errorf("upsert doctor_direction doctor=%s dir=%q: %w", d.SourceID, dirName, err)
			}
			res.DoctorDirs++
		}
	}

	// ── 5. Doctor–service assignments ────────────────────────────────────────
	for _, a := range plan.Assignments {
		if a.MatchConfidence == "unmatched" || a.ServiceCode == "" {
			res.Skipped++
			continue
		}

		doctorID, ok := doctorIDs[a.DoctorSourceID]
		if !ok {
			res.Skipped++
			continue
		}
		serviceID, ok := serviceCodeIDs[a.ServiceCode]
		if !ok {
			res.Skipped++
			continue
		}

		patientType := string(a.PatientType)
		if patientType == "" {
			patientType = "both"
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO doctor_services (doctor_id, service_id, patient_type)
			VALUES ($1, $2, $3::patient_audience)
			ON CONFLICT (doctor_id, service_id)
			DO UPDATE SET patient_type = EXCLUDED.patient_type`,
			doctorID, serviceID, patientType,
		); err != nil {
			return nil, fmt.Errorf("upsert doctor_service doctor=%s service=%s: %w",
				a.DoctorSourceID, a.ServiceCode, err)
		}
		res.DoctorServices++
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return res, nil
}

// nullableString converts an empty string to nil so pgx stores NULL in the DB.
func nullableString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
