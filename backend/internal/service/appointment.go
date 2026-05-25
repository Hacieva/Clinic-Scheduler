package service

import (
	"context"
	"time"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type AppointmentService struct {
	repo          repository.AppointmentRepository
	doctorRepo    repository.DoctorRepository
	svcRepo       repository.ServiceRepository
	doctorSvcRepo repository.DoctorServiceRepository
}

func NewAppointmentService(
	repo repository.AppointmentRepository,
	doctorRepo repository.DoctorRepository,
	svcRepo repository.ServiceRepository,
	doctorSvcRepo repository.DoctorServiceRepository,
) *AppointmentService {
	return &AppointmentService{
		repo:          repo,
		doctorRepo:    doctorRepo,
		svcRepo:       svcRepo,
		doctorSvcRepo: doctorSvcRepo,
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
	StartAt                 time.Time
	Source                  model.AppointmentSource
	PatientComment          *string
	CreatedByUserID         *int64
}

// validTransitions is the single source of truth for the appointment status machine.
var validTransitions = map[model.AppointmentStatus][]model.AppointmentStatus{
	model.StatusCreated: {
		model.StatusConfirmed,
		model.StatusCancelledByPatient,
		model.StatusCancelledByAdmin,
	},
	model.StatusConfirmed: {
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

// Create validates business rules, computes EndAt and DirectionID from the
// service record, then delegates persistence to the repository.
func (s *AppointmentService) Create(ctx context.Context, input CreateAppointmentInput) (*model.Appointment, error) {
	dw, err := s.doctorRepo.GetByID(ctx, input.DoctorID)
	if err != nil {
		return nil, err
	}
	if !dw.IsActive {
		return nil, apperrors.ErrDoctorInactive
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

	var directionID int64
	if svc.DirectionID != nil {
		directionID = *svc.DirectionID
	}

	return s.repo.Create(ctx, repository.CreateAppointmentInput{
		PatientTelegramID:       input.PatientTelegramID,
		PatientTelegramUsername: input.PatientTelegramUsername,
		PatientName:             input.PatientName,
		PatientPhone:            input.PatientPhone,
		DoctorID:                input.DoctorID,
		ServiceID:               input.ServiceID,
		DirectionID:             directionID,
		StartAt:                 input.StartAt,
		EndAt:                   endAt,
		Source:                  input.Source,
		PatientComment:          input.PatientComment,
		CreatedByUserID:         input.CreatedByUserID,
	})
}

func (s *AppointmentService) GetByID(ctx context.Context, id int64) (*repository.AppointmentDetail, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AppointmentService) List(ctx context.Context, filter repository.AppointmentFilter) ([]repository.AppointmentDetail, error) {
	return s.repo.List(ctx, filter)
}

// changeStatus is the single implementation path for all status transitions.
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
	return s.repo.UpdateStatus(ctx, id, detail.Status, to, changedByUserID, comment)
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
