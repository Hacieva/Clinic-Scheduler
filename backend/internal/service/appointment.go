package service

import (
	"context"
	"time"

	"github.com/Hacieva/clinic-scheduler/backend/internal/availability"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// AppointmentScheduleChecker validates that a booking falls within working hours.
// Satisfied by *repository.ScheduleRepo (duck-typed via availability.ScheduleRepository).
type AppointmentScheduleChecker = availability.ScheduleRepository

type AppointmentService struct {
	repo            repository.AppointmentRepository
	visitRepo       repository.VisitRepository
	doctorRepo      repository.DoctorRepository
	svcRepo         repository.ServiceRepository
	doctorSvcRepo   repository.DoctorServiceRepository
	scheduleChecker AppointmentScheduleChecker
}

func NewAppointmentService(
	repo repository.AppointmentRepository,
	visitRepo repository.VisitRepository,
	doctorRepo repository.DoctorRepository,
	svcRepo repository.ServiceRepository,
	doctorSvcRepo repository.DoctorServiceRepository,
	scheduleChecker AppointmentScheduleChecker,
) *AppointmentService {
	return &AppointmentService{
		repo:            repo,
		visitRepo:       visitRepo,
		doctorRepo:      doctorRepo,
		svcRepo:         svcRepo,
		doctorSvcRepo:   doctorSvcRepo,
		scheduleChecker: scheduleChecker,
	}
}

// CreateAppointmentInput is the service-level booking input.
// DirectionID and EndAt are computed internally from the service record.
type CreateAppointmentInput struct {
	PatientTelegramID       *int64
	PatientTelegramUsername *string
	PatientName             string
	PatientPhone            string
	DoctorID                int64
	ServiceID               int64
	BranchID                *int64
	VisitID                 *int64                // nil = auto-create Visit
	AppointmentType         model.AppointmentType // default: scheduled
	StartAt                 time.Time
	Source                  model.AppointmentSource
	PatientComment          *string
	CreatedByUserID         *int64
}

// validTransitions is the single source of truth for the appointment status machine.
// arrived is included because it blocks the slot (patient is physically with the doctor).
var validTransitions = map[model.AppointmentStatus][]model.AppointmentStatus{
	model.StatusCreated: {
		model.StatusConfirmed,
		model.StatusArrived,
		model.StatusCancelledByPatient,
		model.StatusCancelledByAdmin,
	},
	model.StatusConfirmed: {
		model.StatusArrived,
		model.StatusCancelledByAdmin,
	},
	model.StatusArrived: {
		model.StatusCompleted,
		model.StatusNoShow,
		model.StatusCancelledByAdmin,
	},
}

func canTransition(from, to model.AppointmentStatus) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// Create validates business rules, resolves Visit, computes EndAt and DirectionID,
// then delegates persistence to the repository.
func (s *AppointmentService) Create(ctx context.Context, input CreateAppointmentInput) (*model.Appointment, error) {
	apptType := input.AppointmentType
	if apptType == "" {
		apptType = model.AppointmentTypeScheduled
	}

	dw, err := s.doctorRepo.GetByID(ctx, input.DoctorID)
	if err != nil {
		return nil, err
	}
	if !dw.IsActive {
		return nil, apperrors.ErrDoctorInactive
	}

	// Enforce booking_mode: queue_only rejects scheduled; appointment_only rejects walk_in.
	switch dw.BookingMode {
	case "queue_only":
		if apptType == model.AppointmentTypeScheduled {
			return nil, apperrors.ErrInvalidBookingMode
		}
	case "appointment_only":
		if apptType == model.AppointmentTypeWalkIn {
			return nil, apperrors.ErrInvalidBookingMode
		}
	}

	svc, err := s.svcRepo.GetByID(ctx, input.ServiceID)
	if err != nil {
		return nil, err
	}
	if !svc.IsActive {
		return nil, apperrors.ErrNotFound
	}

	// Validate doctor–service assignment via junction table (authoritative source).
	assigned, err := s.doctorSvcRepo.IsAssigned(ctx, input.DoctorID, input.ServiceID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, apperrors.ErrDirectionMismatch
	}

	if !input.StartAt.After(time.Now()) {
		return nil, apperrors.ErrOutsideHours
	}

	endAt := input.StartAt.Add(time.Duration(svc.DurationMinutes) * time.Minute)

	// Time-slot validation only applies to scheduled appointments.
	if apptType == model.AppointmentTypeScheduled {
		if err := s.checkWorkingHours(ctx, input.DoctorID, input.StartAt, endAt); err != nil {
			return nil, err
		}
	}

	// Resolve the effective branch: caller value first, doctor's branch as fallback.
	// The repo uses this to auto-create a Visit atomically inside its transaction
	// when VisitID is nil and appointment_type is 'scheduled'.
	effectiveBranchID := input.BranchID
	if effectiveBranchID == nil && dw.BranchID != nil {
		effectiveBranchID = dw.BranchID
	}

	appt, err := s.repo.Create(ctx, repository.CreateAppointmentInput{
		PatientTelegramID:       input.PatientTelegramID,
		PatientTelegramUsername: input.PatientTelegramUsername,
		PatientName:             input.PatientName,
		PatientPhone:            input.PatientPhone,
		DoctorID:                input.DoctorID,
		ServiceID:               input.ServiceID,
		DirectionID:             svc.DirectionID,
		BranchID:                effectiveBranchID,
		VisitID:                 input.VisitID,
		AppointmentType:         apptType,
		StartAt:                 input.StartAt,
		EndAt:                   endAt,
		Source:                  input.Source,
		PatientComment:          input.PatientComment,
		CreatedByUserID:         input.CreatedByUserID,
	})
	if err != nil {
		return nil, err
	}
	return appt, nil
}

func (s *AppointmentService) GetByID(ctx context.Context, id int64) (*repository.AppointmentDetail, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AppointmentService) List(ctx context.Context, filter repository.AppointmentFilter) ([]repository.AppointmentDetail, error) {
	return s.repo.List(ctx, filter)
}

// Arrive transitions an appointment to 'arrived' and updates the parent Visit to 'in_progress'.
func (s *AppointmentService) Arrive(ctx context.Context, id int64, changedByUserID *int64) error {
	detail, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !canTransition(detail.Status, model.StatusArrived) {
		return apperrors.ErrInvalidStatusTransition
	}
	if err := s.repo.UpdateStatus(ctx, id, detail.Status, model.StatusArrived, changedByUserID, nil); err != nil {
		return err
	}
	if detail.VisitID != nil {
		now := time.Now()
		// Transition visit to in_progress; set arrived_at only if not already set.
		if err := s.visitRepo.UpdateStatus(ctx, *detail.VisitID, model.VisitStatusInProgress, &now, nil); err != nil {
			// Log but don't fail the appointment transition — visit sync is best-effort.
			_ = err
		}
	}
	return nil
}

// changeStatus is the single implementation path for all status transitions.
// After the update it triggers a Visit status sync.
func (s *AppointmentService) changeStatus(
	ctx context.Context,
	id int64,
	to model.AppointmentStatus,
	changedByUserID *int64,
	comment *string,
) error {
	detail, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !canTransition(detail.Status, to) {
		return apperrors.ErrInvalidStatusTransition
	}
	if err := s.repo.UpdateStatus(ctx, id, detail.Status, to, changedByUserID, comment); err != nil {
		return err
	}
	if detail.VisitID != nil {
		_ = s.syncVisitStatus(ctx, *detail.VisitID)
	}
	return nil
}

func (s *AppointmentService) Confirm(ctx context.Context, id int64, changedByUserID *int64) error {
	return s.changeStatus(ctx, id, model.StatusConfirmed, changedByUserID, nil)
}

func (s *AppointmentService) CancelByAdmin(ctx context.Context, id int64, changedByUserID *int64, comment *string) error {
	return s.changeStatus(ctx, id, model.StatusCancelledByAdmin, changedByUserID, comment)
}

func (s *AppointmentService) CancelByPatient(ctx context.Context, id int64) error {
	return s.changeStatus(ctx, id, model.StatusCancelledByPatient, nil, nil)
}

func (s *AppointmentService) Complete(ctx context.Context, id int64, changedByUserID *int64) error {
	return s.changeStatus(ctx, id, model.StatusCompleted, changedByUserID, nil)
}

func (s *AppointmentService) MarkNoShow(ctx context.Context, id int64, changedByUserID *int64) error {
	return s.changeStatus(ctx, id, model.StatusNoShow, changedByUserID, nil)
}

// GetDoctorIDByUserID resolves the doctor record for a given JWT user_id.
func (s *AppointmentService) GetDoctorIDByUserID(ctx context.Context, userID int64) (int64, error) {
	return s.doctorRepo.GetDoctorIDByUserID(ctx, userID)
}

// syncVisitStatus re-derives the Visit status from its appointments and updates if changed.
// Called after every appointment status transition that has a visit_id.
// Errors are non-fatal — logged via the caller's error discard.
func (s *AppointmentService) syncVisitStatus(ctx context.Context, visitID int64) error {
	limit := 100
	appts, err := s.repo.List(ctx, repository.AppointmentFilter{VisitID: &visitID, Limit: limit})
	if err != nil {
		return err
	}
	if len(appts) == 0 {
		return nil
	}

	var (
		hasArrived    bool
		hasCompleted  bool
		hasNoShow     bool
		cancelledCount int
	)
	for _, a := range appts {
		switch a.Status {
		case model.StatusArrived:
			hasArrived = true
		case model.StatusCompleted:
			hasCompleted = true
		case model.StatusNoShow:
			hasNoShow = true
		case model.StatusCancelledByAdmin, model.StatusCancelledByPatient:
			cancelledCount++
		}
	}

	nonCancelled := len(appts) - cancelledCount

	var newStatus model.VisitStatus
	var completedAt *time.Time

	switch {
	case hasArrived:
		newStatus = model.VisitStatusInProgress
	case cancelledCount == len(appts):
		newStatus = model.VisitStatusCancelled
	case nonCancelled > 0 && hasCompleted && !hasArrived && !hasNoShow:
		// All non-cancelled appointments are completed
		allDone := true
		for _, a := range appts {
			if a.Status != model.StatusCompleted &&
				a.Status != model.StatusCancelledByAdmin &&
				a.Status != model.StatusCancelledByPatient {
				allDone = false
				break
			}
		}
		if allDone {
			newStatus = model.VisitStatusCompleted
			now := time.Now()
			completedAt = &now
		} else {
			newStatus = model.VisitStatusInProgress
		}
	case hasNoShow && !hasArrived && !hasCompleted:
		newStatus = model.VisitStatusNoShow
	default:
		newStatus = model.VisitStatusScheduled
	}

	return s.visitRepo.UpdateStatus(ctx, visitID, newStatus, nil, completedAt)
}

// checkWorkingHours returns ErrOutsideHours when [startAt, endAt) does not fall
// within any working interval for that doctor on startAt's calendar day.
func (s *AppointmentService) checkWorkingHours(ctx context.Context, doctorID int64, startAt, endAt time.Time) error {
	day := time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 0, 0, 0, 0, startAt.Location())

	schedule, err := s.scheduleChecker.GetWorkingHours(ctx, doctorID)
	if err != nil {
		return err
	}

	exceptions, err := s.scheduleChecker.GetScheduleExceptions(ctx, doctorID, day, day)
	if err != nil {
		return err
	}

	if !availability.IsWithinWorkingHours(startAt, endAt, availability.CalculatorInput{
		Date:            day,
		RegularSchedule: schedule,
		Exceptions:      exceptions,
	}) {
		return apperrors.ErrOutsideHours
	}
	return nil
}
