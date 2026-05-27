package importer

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ParseConfig holds the file paths for all three source files.
type ParseConfig struct {
	DoctorsPath   string // График_врачей_МЕДИК_ПРОФИ.xlsx (required)
	PricesPath    string // Медлок_Прайсы_и_Врачи_Второй_Филиал.xlsx (required)
	DikiDiPath    string // Новая таблица.xlsx (optional; used for duration cross-check)
	OverridesPath string // manual_overrides.csv (optional)
}

// Parse reads all source files and returns an ImportPlan.
// No database connections are opened. Safe to run repeatedly.
func Parse(cfg ParseConfig) (*ImportPlan, error) {
	plan := &ImportPlan{}

	// Branches are known in advance from clinic data.
	plan.Branches = []BranchRow{
		{Name: "Главный филиал", Address: "В.В. Путина 17А"},
		{Name: "Второй филиал", Address: ""},
	}

	if err := parseDoctors(cfg.DoctorsPath, plan); err != nil {
		return nil, fmt.Errorf("parsing doctors: %w", err)
	}

	if err := parseServices(cfg.PricesPath, plan); err != nil {
		return nil, fmt.Errorf("parsing services: %w", err)
	}

	overrides, err := loadOverrides(cfg.OverridesPath)
	if err != nil {
		return nil, fmt.Errorf("loading overrides: %w", err)
	}

	if err := parseAssignments(cfg.DoctorsPath, plan, overrides); err != nil {
		return nil, fmt.Errorf("parsing assignments: %w", err)
	}

	return plan, nil
}

// parseDoctors reads the doctors_v2 and Приезжающие sheets.
func parseDoctors(path string, plan *ImportPlan) error {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	directionSet := map[string]bool{}

	// ---- doctors_v2 sheet ----
	rows, err := f.GetRows("doctors_v2")
	if err != nil {
		return fmt.Errorf("sheet doctors_v2: %w", err)
	}
	if len(rows) < 2 {
		return fmt.Errorf("doctors_v2 sheet has no data rows")
	}

	colIdx := headerIndex(rows[0])

	// Day columns: column name → ISO day-of-week (1=Mon … 6=Sat)
	dayCols := []struct {
		col string
		dow int
	}{
		{"mon", 1}, {"tue", 2}, {"wed", 3}, {"thu", 4}, {"fri", 5}, {"sat", 6},
	}

	for i, row := range rows[1:] {
		cell := cellGetter(row, colIdx)

		sourceID := cell("id")
		if sourceID == "" {
			continue
		}

		fullName := cell("full_name")
		if fullName == "" {
			plan.warn("doctor", sourceID, "missing_name",
				fmt.Sprintf("row %d has no full_name — skipped", i+2))
			continue
		}

		last, first, middle := SplitName(fullName)

		kind, mode, unresolved := MapBookingType(cell("booking_type"))
		if unresolved {
			plan.warn("doctor", sourceID, "unresolved_booking_mode",
				fmt.Sprintf("booking_type=%q → defaulted to staff/appointment_only", cell("booking_type")))
		}

		audience := MapAudience(cell("audience"))

		var wh []WorkingHoursRow
		for _, dc := range dayCols {
			parsed, err := ParseWorkingHours(cell(dc.col), dc.dow)
			if err != nil {
				plan.warn("doctor", sourceID, "unresolved_schedule",
					fmt.Sprintf("day %d: %v", dc.dow, err))
				continue
			}
			wh = append(wh, parsed...)
		}

		dirs := ParseDirections(cell("specialty"))
		for _, d := range dirs {
			directionSet[d] = true
		}

		plan.Doctors = append(plan.Doctors, DoctorRow{
			SourceID:     sourceID,
			FullName:     fullName,
			FirstName:    first,
			LastName:     last,
			MiddleName:   middle,
			BranchName:   mapBranchName(cell("branch")),
			Directions:   dirs,
			Audience:     audience,
			DoctorKind:   kind,
			BookingMode:  mode,
			IsActive:     true,
			WorkingHours: wh,
		})
	}

	// ---- Приезжающие sheet (visiting doctors) ----
	visitingRows, err := f.GetRows("Приезжающие (раз в 2 мес)")
	if err != nil {
		plan.warn("doctor", "visiting", "sheet_not_found",
			"sheet 'Приезжающие (раз в 2 мес)' not found — visiting doctors skipped")
	} else {
		for i, row := range visitingRows[1:] {
			if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
				continue
			}
			fullName := strings.TrimSpace(row[0])
			specialty := ""
			if len(row) > 1 {
				specialty = strings.TrimSpace(row[1])
			}

			sourceID := fmt.Sprintf("V%03d", i+1)
			last, first, middle := SplitName(fullName)
			dirs := ParseDirections(specialty)
			for _, d := range dirs {
				directionSet[d] = true
			}

			plan.Doctors = append(plan.Doctors, DoctorRow{
				SourceID:    sourceID,
				FullName:    fullName,
				FirstName:   first,
				LastName:    last,
				MiddleName:  middle,
				BranchName:  "", // visiting doctors are not branch-specific
				Directions:  dirs,
				Audience:    nil,
				DoctorKind:  DoctorKindVisiting,
				BookingMode: BookingModeAppointmentOnly,
				IsActive:    true,
			})
		}
	}

	// Collect unique directions.
	for name := range directionSet {
		plan.Directions = append(plan.Directions, DirectionRow{Name: name})
	}

	return nil
}

// parseServices reads both Medlock price sheets.
// Services are inserted into the global catalog once (deduped by code or name).
// Category is derived from the section header row immediately preceding each service.
func parseServices(path string, plan *ImportPlan) error {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	// Iterate sheets in deterministic order; Главный is primary for price.
	sheets := []struct {
		name       string
		branchName string
	}{
		{"ПРАЙС ГЛАВНОГО ФИЛИАЛА", "Главный филиал"},
		{"ПРАЙС ВТОРОГО ФИЛИАЛА", "Второй филиал"},
	}

	// dedup key → true: code if present, else lowercased name
	seen := map[string]bool{}

	for _, sh := range sheets {
		rows, err := f.GetRows(sh.name)
		if err != nil {
			plan.warn("service", sh.name, "sheet_not_found",
				fmt.Sprintf("sheet %q not found — skipped", sh.name))
			continue
		}
		if len(rows) < 2 {
			continue
		}

		colIdx := headerIndex(rows[0])
		nameCol, hasName := colIdx["название услуги"]
		priceCol, hasPrice := colIdx["цена"]
		codeCol, hasCode := colIdx["код услуги"]
		durCol, hasDur := colIdx["длительность"]

		if !hasName {
			return fmt.Errorf("sheet %q: missing column 'Название услуги'", sh.name)
		}

		currentCategory := "Прочее"

		for _, row := range rows[1:] {
			get := func(idx int, present bool) string {
				if !present || idx >= len(row) {
					return ""
				}
				return strings.TrimSpace(row[idx])
			}

			name := get(nameCol, hasName)
			if name == "" {
				continue
			}
			price := get(priceCol, hasPrice)
			code := get(codeCol, hasCode)
			dur := get(durCol, hasDur)

			// Section header: no code and no price → update current category.
			if code == "" && price == "" {
				currentCategory = name
				continue
			}

			durMin, durOK := ParseDuration(dur)
			if !durOK {
				plan.warn("service", name, "missing_duration",
					fmt.Sprintf("code=%q raw=%q", code, dur))
			}

			priceKopecks, priceOK := ParsePriceKopecks(price)
			if !priceOK {
				plan.warn("service", name, "missing_price",
					fmt.Sprintf("code=%q raw=%q", code, price))
			}

			// Dedup: primary key is code; fallback is lowercased name.
			dedupKey := code
			if dedupKey == "" {
				dedupKey = "__name__" + strings.ToLower(name)
			}
			if seen[dedupKey] {
				continue
			}
			seen[dedupKey] = true

			plan.Services = append(plan.Services, ServiceRow{
				Code:            code,
				Name:            name,
				Category:        currentCategory,
				DurationMinutes: durMin,
				Price:           priceKopecks,
				BranchName:      sh.branchName,
			})
		}
	}

	return nil
}

// parseAssignments reads doctor_services_v2 and matches each row to parsed
// doctors and services.
func parseAssignments(path string, plan *ImportPlan, overrides map[string]string) error {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	// Build lookup: normalized full name → sourceID
	doctorByName := make(map[string]string, len(plan.Doctors))
	for _, d := range plan.Doctors {
		key := strings.ToLower(strings.TrimSpace(d.FullName))
		doctorByName[key] = d.SourceID
	}

	// Build lookup: normalized service name → code
	serviceByNorm := make(map[string]string, len(plan.Services))
	for _, s := range plan.Services {
		norm := NormalizeServiceName(s.Name)
		if norm != "" {
			serviceByNorm[norm] = s.Code
		}
	}

	rows, err := f.GetRows("doctor_services_v2")
	if err != nil {
		return fmt.Errorf("sheet doctor_services_v2: %w", err)
	}
	if len(rows) < 2 {
		return nil
	}

	colIdx := headerIndex(rows[0])

	for _, row := range rows[1:] {
		cell := cellGetter(row, colIdx)

		doctorName := cell("doctor_name")
		serviceName := cell("service_name")
		if doctorName == "" || serviceName == "" {
			continue
		}

		sourceID, found := doctorByName[strings.ToLower(doctorName)]
		if !found {
			plan.warn("assignment", doctorName, "unmatched_doctor",
				fmt.Sprintf("doctor %q not in doctors_v2 — assignment skipped", doctorName))
			continue
		}

		normName := NormalizeServiceName(serviceName)
		serviceCode, confidence := matchService(normName, serviceByNorm, overrides, serviceName)
		if confidence == "unmatched" {
			plan.warn("assignment", serviceName, "unmatched_service",
				fmt.Sprintf("doctor=%s normalized=%q — add to manual_overrides.csv", doctorName, normName))
		}

		patientType := MapPatientType(cell("patient_type"))
		price, _ := ParsePriceKopecks(cell("price"))

		plan.Assignments = append(plan.Assignments, AssignmentRow{
			DoctorSourceID:  sourceID,
			DoctorName:      doctorName,
			ServiceName:     serviceName,
			ServiceCode:     serviceCode,
			PatientType:     patientType,
			Price:           price,
			MatchConfidence: confidence,
		})
	}

	return nil
}

// matchService resolves a normalized service name to a canonical code.
// Priority: manual override → exact normalized match → basic fuzzy match.
func matchService(norm string, serviceByNorm, overrides map[string]string, rawName string) (code, confidence string) {
	// 1. Manual override (highest priority).
	if c, ok := overrides[strings.ToLower(strings.TrimSpace(rawName))]; ok {
		return c, "override"
	}
	// 2. Exact normalized match.
	if c, ok := serviceByNorm[norm]; ok {
		return c, "exact"
	}
	// 3. Basic fuzzy: one normalized name contains the other.
	// Only match if both strings are reasonably long (≥8 chars) to avoid noise.
	if len([]rune(norm)) >= 8 {
		for svcNorm, svcCode := range serviceByNorm {
			if len([]rune(svcNorm)) >= 8 {
				if strings.Contains(svcNorm, norm) || strings.Contains(norm, svcNorm) {
					return svcCode, "fuzzy"
				}
			}
		}
	}
	return "", "unmatched"
}

// loadOverrides reads manual_overrides.csv (optional).
// Returns an empty map if the file does not exist.
func loadOverrides(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}
	fh, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("opening overrides %s: %w", path, err)
	}
	defer fh.Close()

	r := csv.NewReader(fh)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading overrides CSV: %w", err)
	}

	overrides := make(map[string]string, len(records))
	for i, rec := range records {
		if i == 0 {
			continue // skip header
		}
		if len(rec) < 2 {
			continue
		}
		rawName := strings.ToLower(strings.TrimSpace(rec[0]))
		code := strings.TrimSpace(rec[1])
		if rawName != "" && code != "" {
			overrides[rawName] = code
		}
	}
	return overrides, nil
}

// headerIndex builds a column-name → index map from a header row.
// Keys are lowercased and trimmed.
func headerIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return idx
}

// cellGetter returns a function that reads a cell value by column name.
func cellGetter(row []string, colIdx map[string]int) func(name string) string {
	return func(name string) string {
		i, ok := colIdx[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
}

// mapBranchName normalizes raw branch strings from the source spreadsheet.
func mapBranchName(raw string) string {
	switch strings.TrimSpace(raw) {
	case "Второй филиал", "Второй", "второй":
		return "Второй филиал"
	case "Главный филиал", "Главный", "главный":
		return "Главный филиал"
	default:
		return strings.TrimSpace(raw)
	}
}
