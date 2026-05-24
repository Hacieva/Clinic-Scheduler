package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Hacieva/clinic-scheduler/backend/internal/api/handler"
	"github.com/Hacieva/clinic-scheduler/backend/internal/api/middleware"
	"github.com/Hacieva/clinic-scheduler/backend/internal/availability"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

func main() {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET is not set")
		os.Exit(1)
	}

	botSecret := os.Getenv("BOT_API_SECRET")
	if botSecret == "" {
		slog.Warn("BOT_API_SECRET is not set — bot endpoints will reject all requests")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepo(pool)
	authSvc := service.NewAuthService(userRepo, jwtSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	directionRepo := repository.NewDirectionRepo(pool)
	directionSvc := service.NewDirectionService(directionRepo)
	directionHandler := handler.NewDirectionHandler(directionSvc)

	doctorRepo := repository.NewDoctorRepo(pool)
	doctorSvc := service.NewDoctorService(doctorRepo, directionRepo)
	doctorHandler := handler.NewDoctorHandler(doctorSvc)

	serviceRepo := repository.NewServiceRepo(pool)

	// Legacy per-doctor service management (still used by bot and old UI paths).
	// TODO: migrate bot to use doctor_services junction endpoints.
	medicalSvc := service.NewMedicalServiceService(serviceRepo, doctorRepo)
	serviceHandler := handler.NewServiceHandler(medicalSvc)

	// Global catalog: services are owned by the clinic, not individual doctors.
	catalogSvc := service.NewServiceCatalogService(serviceRepo)
	catalogHandler := handler.NewServiceCatalogHandler(catalogSvc)

	// Doctor–service assignment via junction table.
	doctorSvcRepo := repository.NewDoctorServiceRepo(pool)
	assignmentSvc := service.NewDoctorAssignmentService(doctorSvcRepo, serviceRepo, doctorRepo)
	assignmentHandler := handler.NewDoctorAssignmentHandler(assignmentSvc)

	scheduleRepo := repository.NewScheduleRepo(pool)
	scheduleSvc := service.NewScheduleService(scheduleRepo)
	scheduleHandler := handler.NewScheduleHandler(scheduleSvc)

	apptSlotRepo := repository.NewAppointmentSlotRepo(pool)
	availSvc := availability.NewService(scheduleRepo, apptSlotRepo, serviceRepo)
	availHandler := handler.NewAvailabilityHandler(availSvc)

	branchRepo := repository.NewBranchRepo(pool)
	branchSvc := service.NewBranchService(branchRepo)
	branchHandler := handler.NewBranchHandler(branchSvc)

	patientRepo := repository.NewPatientRepo(pool)
	patientSvc := service.NewPatientService(patientRepo)
	patientHandler := handler.NewPatientHandler(patientSvc)

	apptRepo := repository.NewAppointmentRepo(pool)
	apptSvc := service.NewAppointmentService(apptRepo, doctorRepo, serviceRepo, doctorSvcRepo)
	apptHandler := handler.NewAppointmentHandler(apptSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		loginLimiter := middleware.LoginRateLimit(5, time.Minute)
		r.With(loginLimiter).Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)

		// Bot endpoints — X-Bot-Token auth, no JWT
		// TODO: add GET /bot/doctors/{id}/assigned-services after bot migration.
		r.Group(func(r chi.Router) {
			r.Use(middleware.BotAuth(botSecret))
			r.Get("/bot/directions", directionHandler.List)
			r.Get("/bot/doctors", doctorHandler.List)
			r.Get("/bot/doctors/{id}/services", serviceHandler.List) // TODO: legacy; uses doctor_id column
			r.Post("/bot/appointments", apptHandler.BotCreate)
			r.Post("/bot/appointments/{id}/cancel", apptHandler.BotCancel)
			r.Get("/bot/availability", availHandler.GetAvailability)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(jwtSecret))
			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me", authHandler.Me)
			r.Post("/auth/change-password", authHandler.ChangePassword)

			r.Get("/directions", directionHandler.List)
			r.Get("/directions/{id}", directionHandler.GetByID)
			r.Get("/doctors", doctorHandler.List)
			r.Get("/doctors/{id}", doctorHandler.GetByID)

			// Legacy per-doctor service endpoints (kept for backward compat).
			// TODO: remove after frontend fully migrates to /assigned-services.
			r.Get("/doctors/{id}/services", serviceHandler.List)
			r.Get("/doctors/{id}/services/{serviceId}", serviceHandler.GetByID)

			// Junction-based assignment (new authoritative endpoints).
			r.Get("/doctors/{id}/assigned-services", assignmentHandler.ListAssigned)

			r.Get("/doctors/{id}/working-hours", scheduleHandler.ListWorkingHours)
			r.Get("/doctors/{id}/exceptions", scheduleHandler.ListExceptions)

			// Global service catalog (all authenticated users can read).
			r.Get("/services", catalogHandler.ListAll)

			r.Get("/availability", availHandler.GetAvailability)

			// Branch endpoints — owner + admin
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("owner", "admin"))
				r.Get("/branches", branchHandler.List)
				r.Get("/branches/{id}", branchHandler.GetByID)
				r.Post("/branches", branchHandler.Create)
				r.Patch("/branches/{id}", branchHandler.Update)
				r.Delete("/branches/{id}", branchHandler.Delete)
			})

			// Patient endpoints — owner + admin
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("owner", "admin"))
				r.Get("/patients", patientHandler.List)
				r.Get("/patients/{id}", patientHandler.GetByID)
				r.Post("/patients", patientHandler.Create)
				r.Patch("/patients/{id}", patientHandler.Update)
			})

			// Doctor-only appointment routes (privacy trimmed, doctor_id from JWT)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("doctor"))
				r.Get("/doctor/appointments", apptHandler.DoctorList)
				r.Get("/doctor/appointments/{id}", apptHandler.DoctorGetByID)
			})

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin", "owner"))
				r.Post("/directions", directionHandler.Create)
				r.Put("/directions/{id}", directionHandler.Update)
				r.Delete("/directions/{id}", directionHandler.Delete)
				r.Post("/doctors", doctorHandler.Create)
				r.Patch("/doctors/{id}", doctorHandler.Update)
				r.Delete("/doctors/{id}", doctorHandler.Delete)
				r.Post("/doctors/{id}/account", doctorHandler.CreateAccount)
				r.Put("/doctors/{id}/directions", doctorHandler.SetDirections)

				// Legacy per-doctor service CRUD (kept for backward compat).
				// TODO: remove after frontend migrates to catalog + assignment flow.
				r.Post("/doctors/{id}/services", serviceHandler.Create)
				r.Put("/doctors/{id}/services/{serviceId}", serviceHandler.Update)
				r.Delete("/doctors/{id}/services/{serviceId}", serviceHandler.Delete)

				// Junction-based assignment mutations (new authoritative endpoints).
				r.Put("/doctors/{id}/assigned-services", assignmentHandler.BulkSet)
				r.Post("/doctors/{id}/assigned-services/{serviceId}", assignmentHandler.Assign)
				r.Delete("/doctors/{id}/assigned-services/{serviceId}", assignmentHandler.Unassign)

				r.Put("/doctors/{id}/working-hours", scheduleHandler.ReplaceWorkingHours)
				r.Post("/doctors/{id}/exceptions", scheduleHandler.CreateException)
				r.Post("/doctors/{id}/exceptions/range", scheduleHandler.CreateExceptionRange)
				r.Put("/doctors/{id}/exceptions/{exId}", scheduleHandler.UpdateException)
				r.Delete("/doctors/{id}/exceptions/{exId}", scheduleHandler.DeleteException)

				// Global service catalog mutations.
				r.Post("/services", catalogHandler.Create)
				r.Put("/services/{id}", catalogHandler.Update)
				r.Delete("/services/{id}", catalogHandler.Delete)

				r.Post("/appointments", apptHandler.AdminCreate)
				r.Get("/appointments", apptHandler.List)
				r.Get("/appointments/{id}", apptHandler.GetByID)
				r.Post("/appointments/{id}/confirm", apptHandler.Confirm)
				r.Post("/appointments/{id}/cancel", apptHandler.AdminCancel)
				r.Post("/appointments/{id}/complete", apptHandler.Complete)
				r.Post("/appointments/{id}/no-show", apptHandler.MarkNoShow)
			})
		})
	})

	slog.Info("server starting", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
