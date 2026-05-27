package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Hacieva/clinic-scheduler/backend/internal/importer"
)

func main() {
	var (
		doctorsPath     = flag.String("doctors", "", "Path to График_врачей_МЕДИК_ПРОФИ.xlsx (required)")
		pricesPath      = flag.String("prices", "", "Path to Медлок_Прайсы_и_Врачи_Второй_Филиал.xlsx (required)")
		dikiDiPath      = flag.String("dikidi", "", "Path to Новая таблица.xlsx (optional)")
		overridesPath   = flag.String("overrides", "", "Path to manual_overrides.csv (optional)")
		syntheticPath   = flag.String("synthetic", "", "Path to synthetic_services.csv (optional)")
		branchFilter    = flag.String("branch", "", "Filter output by branch name (optional)")
		dbURL           = flag.String("db", "", "PostgreSQL connection URL (defaults to DATABASE_URL env var)")
		dumpServices    = flag.Bool("dump-services", false, "Print parsed service catalog as CSV and exit")
		dumpAssignments = flag.Bool("dump-assignments", false, "Print all parsed assignments as CSV and exit")
		doImport        = flag.Bool("import", false, "Execute import (requires --confirm)")
		confirm         = flag.Bool("confirm", false, "Confirm destructive import (required with --import)")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
	}

	if *doctorsPath == "" || *pricesPath == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --doctors and --prices are required")
		flag.Usage()
		os.Exit(1)
	}

	if *doImport && !*confirm {
		fmt.Fprintln(os.Stderr, "ERROR: --import requires --confirm")
		os.Exit(1)
	}

	cfg := importer.ParseConfig{
		DoctorsPath:   *doctorsPath,
		PricesPath:    *pricesPath,
		DikiDiPath:    *dikiDiPath,
		OverridesPath: *overridesPath,
		SyntheticPath: *syntheticPath,
	}

	plan, err := importer.Parse(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		fmt.Fprintln(os.Stderr, "\n=== PARSE FAILED ===")
		os.Exit(1)
	}

	if *dumpServices {
		fmt.Println("code,name,category,duration_minutes,price_kopecks,branch")
		for _, s := range plan.Services {
			fmt.Printf("%q,%q,%q,%d,%d,%q\n",
				s.Code, s.Name, s.Category, s.DurationMinutes, s.Price, s.BranchName)
		}
		return
	}

	if *dumpAssignments {
		fmt.Println("doctor_name,service_name,service_code,patient_type,confidence")
		for _, a := range plan.Assignments {
			fmt.Printf("%q,%q,%q,%q,%q\n",
				a.DoctorName, a.ServiceName, a.ServiceCode, string(a.PatientType), a.MatchConfidence)
		}
		return
	}

	if *branchFilter != "" {
		plan = filterByBranch(plan, *branchFilter)
	}

	if *doImport {
		if *dbURL == "" {
			fmt.Fprintln(os.Stderr, "ERROR: --import requires --db or DATABASE_URL env var")
			os.Exit(1)
		}
		ctx := context.Background()
		db, err := pgxpool.New(ctx, *dbURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: connecting to database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		if err := db.Ping(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: pinging database: %v\n", err)
			os.Exit(1)
		}

		res, err := importer.Execute(ctx, db, plan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: import failed: %v\n", err)
			printSummary(plan, false)
			os.Exit(1)
		}

		fmt.Println("=== IMPORT COMPLETE ===")
		fmt.Println()
		fmt.Printf("Branches written:        %d\n", res.Branches)
		fmt.Printf("Directions written:      %d\n", res.Directions)
		fmt.Printf("Services written:        %d\n", res.Services)
		fmt.Printf("Doctors written:         %d\n", res.Doctors)
		fmt.Printf("Working hour rows:       %d\n", res.WorkingHourRows)
		fmt.Printf("Doctor–direction links:  %d\n", res.DoctorDirs)
		fmt.Printf("Doctor–service links:    %d\n", res.DoctorServices)
		fmt.Printf("Assignments skipped:     %d (unmatched or uncoded)\n", res.Skipped)
		fmt.Println()
		fmt.Println("=== TRANSACTION COMMITTED ===")
		return
	}

	printSummary(plan, true)
}

func filterByBranch(plan *importer.ImportPlan, branch string) *importer.ImportPlan {
	filtered := &importer.ImportPlan{
		Branches:   plan.Branches,
		Directions: plan.Directions,
		Warnings:   plan.Warnings,
	}

	branchDoctors := map[string]bool{}
	for _, d := range plan.Doctors {
		if strings.EqualFold(d.BranchName, branch) || d.BranchName == "" {
			filtered.Doctors = append(filtered.Doctors, d)
			branchDoctors[d.SourceID] = true
		}
	}
	filtered.Services = plan.Services
	for _, a := range plan.Assignments {
		if branchDoctors[a.DoctorSourceID] {
			filtered.Assignments = append(filtered.Assignments, a)
		}
	}
	return filtered
}

func printSummary(plan *importer.ImportPlan, isDryRun bool) {
	if isDryRun {
		fmt.Println("=== DRY RUN — no changes written ===")
	} else {
		fmt.Println("=== IMPORT MODE ===")
	}
	fmt.Println()

	// Aggregate counts from warnings.
	warningCounts := map[string]int{}
	for _, w := range plan.Warnings {
		warningCounts[w.Kind]++
	}

	// Doctor stats.
	whRows := 0
	visitingDoctors := 0
	for _, d := range plan.Doctors {
		whRows += len(d.WorkingHours)
		if d.DoctorKind == importer.DoctorKindVisiting {
			visitingDoctors++
		}
	}

	// Assignment stats.
	unmatchedSvc := 0
	unmatchedDoc := warningCounts["unmatched_doctor"]
	fuzzyMatches := 0
	for _, a := range plan.Assignments {
		switch a.MatchConfidence {
		case "unmatched":
			unmatchedSvc++
		case "fuzzy":
			fuzzyMatches++
		}
	}

	uncatServices := 0
	for _, s := range plan.Services {
		if s.Category == "Прочее" {
			uncatServices++
		}
	}

	// Print counts table.
	fmt.Printf("Branches:       %d parsed\n", len(plan.Branches))
	fmt.Printf("Directions:     %d parsed\n", len(plan.Directions))

	doctorLine := fmt.Sprintf("Doctors:        %d parsed", len(plan.Doctors))
	if visitingDoctors > 0 {
		doctorLine += fmt.Sprintf(",  %d visiting", visitingDoctors)
	}
	if warningCounts["unresolved_booking_mode"] > 0 {
		doctorLine += fmt.Sprintf(",  %d unresolved_booking_mode", warningCounts["unresolved_booking_mode"])
	}
	fmt.Println(doctorLine)

	svcLine := fmt.Sprintf("Services:       %d parsed", len(plan.Services))
	if uncatServices > 0 {
		svcLine += fmt.Sprintf(",  %d → category 'Прочее'", uncatServices)
	}
	if warningCounts["missing_duration"] > 0 {
		svcLine += fmt.Sprintf(",  %d missing_duration", warningCounts["missing_duration"])
	}
	fmt.Println(svcLine)

	whLine := fmt.Sprintf("Working Hours:  %d rows parsed", whRows)
	if warningCounts["unresolved_schedule"] > 0 {
		whLine += fmt.Sprintf(",  %d unresolved_schedule (skipped)", warningCounts["unresolved_schedule"])
	}
	fmt.Println(whLine)

	asnLine := fmt.Sprintf("Assignments:    %d parsed", len(plan.Assignments))
	if unmatchedSvc > 0 {
		asnLine += fmt.Sprintf(",  %d unmatched_service (skipped)", unmatchedSvc)
	}
	if unmatchedDoc > 0 {
		asnLine += fmt.Sprintf(",  %d unmatched_doctor (skipped)", unmatchedDoc)
	}
	if fuzzyMatches > 0 {
		asnLine += fmt.Sprintf(",  %d fuzzy_match (verify manually)", fuzzyMatches)
	}
	fmt.Println(asnLine)

	// Print all warnings grouped by kind.
	if len(plan.Warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings:")

		byKind := map[string][]importer.ImportWarning{}
		for _, w := range plan.Warnings {
			byKind[w.Kind] = append(byKind[w.Kind], w)
		}
		kinds := make([]string, 0, len(byKind))
		for k := range byKind {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		for _, kind := range kinds {
			for _, w := range byKind[kind] {
				fmt.Printf("  [%s %s] %s: %s\n", w.Entity, w.SourceID, w.Kind, w.Detail)
			}
		}
	}

	// Print fuzzy matches for manual review.
	fuzzyList := collectByConfidence(plan, "fuzzy")
	if len(fuzzyList) > 0 {
		fmt.Println()
		fmt.Println("Fuzzy matches (verify manually — source_name → matched_code):")
		for _, entry := range fuzzyList {
			fmt.Printf("  %q → %s\n", entry[0], entry[1])
		}
	}

	// Print unmatched services list (deduped, sorted) for manual_overrides.csv.
	unmatchedNames := collectUnmatched(plan)
	if len(unmatchedNames) > 0 {
		fmt.Println()
		fmt.Println("Unmatched services (add to manual_overrides.csv to resolve):")
		for _, n := range unmatchedNames {
			fmt.Printf("  %q\n", n)
		}
	}

	fmt.Println()
	if isDryRun {
		fmt.Println("=== DRY RUN COMPLETE — review warnings before running --import --confirm ===")
	} else {
		fmt.Println("=== TRANSACTION COMMITTED ===")
	}
}

func collectUnmatched(plan *importer.ImportPlan) []string {
	seen := map[string]bool{}
	for _, a := range plan.Assignments {
		if a.MatchConfidence == "unmatched" && !seen[a.ServiceName] {
			seen[a.ServiceName] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	result := make([]string, 0, len(seen))
	for n := range seen {
		result = append(result, n)
	}
	sort.Strings(result)
	return result
}

// collectByConfidence returns deduped [sourceName, serviceCode] pairs for the given confidence level.
func collectByConfidence(plan *importer.ImportPlan, confidence string) [][2]string {
	seen := map[string]string{}
	for _, a := range plan.Assignments {
		if a.MatchConfidence == confidence {
			seen[a.ServiceName] = a.ServiceCode
		}
	}
	if len(seen) == 0 {
		return nil
	}
	result := make([][2]string, 0, len(seen))
	for name, code := range seen {
		result = append(result, [2]string{name, code})
	}
	sort.Slice(result, func(i, j int) bool { return result[i][0] < result[j][0] })
	return result
}
