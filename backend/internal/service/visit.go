package service

import (
	"context"
	"time"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// VisitService handles visit lifecycle operations.
type VisitService struct {
	visitRepo     repository.VisitRepository
	apptRepo      repository.AppointmentRepository
	patientRepo   repository.PatientRepository
	doctorRepo    repository.DoctorRepository
	svcRepo       repository.ServiceRepository
	doctorSvcRepo repository.DoctorServiceRepository
}

func NewVisitService(
	visitRepo repository.VisitRepository,
	apptRepo repository.AppointmentRepository,
	patientRepo repository.PatientRepository,
	doctorRepo repository.DoctorRepository,
	svcRepo repository.ServiceRepository,
	doctorSvcRepo repository.DoctorServiceRepository,
) *VisitService {
	return &VisitService{
		visitRepo:     visitRepo,
		apptRepo:      apptRepo,
		patientRepo:   patientRepo,
		doctorRepo:    doctorRepo,
		svcRepo:       svcRepo,
		doctorSvcRepo: doctorSvcRepo,
	}
}

// RegisterWalkInInput carries the data for a walk-in patient registration.
type RegisterWalkInInput struct {
	PatientName     string
	PatientPhone    string
	DoctorID        int64
	ServiceID       int64
	BranchID        int64
	Source          model.AppointmentSource
	PatientComment  *string
	Comment         *string // visit-level comment
	CreatedByUserID *int64
}

// RegisterWalkIn creates a Visit (walk_in, in_progress) and a linked Appointment
// (walk_in, arrived, start_at = now) for a patient arriving without a booking.
//
// Patient is looked up by phone first; created if not found.
// The Visit is created before the Appointment with the correct patient_id.
// If the Appointment insert fails, the orphaned Visit is a known MVP limitation.
func (s *VisitService) RegisterWalkIn(ctx context.Context, input RegisterWalkInInput) (*model.Visit, *model.Appointment, error) {
	if input.PatientName == "" || input.PatientPhone == "" {
		return nil, nil, apperrors.ErrInvalidInput
	}

	dw, err := s.doctorRepo.GetByID(ctx, input.DoctorID)
	if err != nil {
		return nil, nil, err
	}
	if !dw.IsActive {
		return nil, nil, apperrors.ErrDoctorInactive
	}
	if dw.BookingMode == "appointment_only" {
		return nil, nil, apperrors.ErrInvalidBookingMode
	}

	svc, err := s.svcRepo.GetByID(ctx, input.ServiceID)
	if err != nil {
		return nil, nil, err
	}
	if !svc.IsActive {
		return nil, nil, apperrors.ErrNotFound
	}

	assigned, err := s.doctorSvcRepo.IsAssigned(ctx, input.DoctorID, input.ServiceID)
	if err != nil {
		return nil, nil, err
	}
	if !assigned {
		return nil, nil, apperrors.ErrDirectionMismatch
	}

	// Resolve patient by phone (upsert semantics: return existing or create new).
	patient, err := s.patientRepo.GetByPhone(ctx, input.PatientPhone)
	if err != nil {
		if !isNotFound(err) {
			return nil, nil, err
		}
		// Patient not found — create them.
		patient, err = s.patientRepo.Create(ctx, repository.CreatePatientInput{
			FullName: input.PatientName,
			Phone:    input.PatientPhone,
			Source:   "admin_panel",
		})
		if err != nil {
			return nil, nil, err
		}
	}

	branchID := input.BranchID
	if branchID == 0 && dw.BranchID != nil {
		branchID = *dw.BranchID
	}

	now := time.Now()

	// Create Visit with the resolved patient_id.
	visit, err := s.visitRepo.Create(ctx, repository.CreateVisitInput{
		PatientID: patient.ID,
		BranchID:  branchID,
		VisitType: model.VisitTypeWalkIn,
		Status:    model.VisitStatusInProgress,
		ArrivedAt: &now,
		Comment:   input.Comment,
	})
	if err != nil {
		return nil, nil, err
	}

	// Create walk-in Appointment (arrived immediately, start_at = now).
	endAt := now.Add(time.Duration(svc.DurationMinutes) * time.Minute)
	appt, err := s.apptRepo.Create(ctx, repository.CreateAppointmentInput{
		PatientID:       &patient.ID, // pre-resolved; skips patient upsert in repo transaction
		PatientName:     input.PatientName,
		PatientPhone:    input.PatientPhone,
		DoctorID:        input.DoctorID,
		ServiceID:       input.ServiceID,
		DirectionID:     svc.DirectionID,
		BranchID:        &branchID,
		VisitID:         &visit.ID,
		AppointmentType: model.AppointmentTypeWalkIn,
		StartAt:         now,
		EndAt:           endAt,
		Source:          input.Source,
		PatientComment:  input.PatientComment,
		CreatedByUserID: input.CreatedByUserID,
	})
	if err != nil {
		return nil, nil, err
	}

	return visit, appt, nil
}

// GetByID returns a single visit by primary key.
func (s *VisitService) GetByID(ctx context.Context, id int64) (*model.Visit, error) {
	return s.visitRepo.GetByID(ctx, id)
}

// List returns visits matching the filter.
func (s *VisitService) List(ctx context.Context, filter repository.VisitFilter) ([]model.Visit, error) {
	return s.visitRepo.List(ctx, filter)
}

// isNotFound returns true when err is a "not found" domain error.
func isNotFound(err error) bool {
	return err != nil && err.Error() == apperrors.ErrNotFound.Error()
}
