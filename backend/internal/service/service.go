package service

import (
	"context"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type MedicalServiceService struct {
	repo    repository.ServiceRepository
	docRepo repository.DoctorRepository
}

func NewMedicalServiceService(repo repository.ServiceRepository, docRepo repository.DoctorRepository) *MedicalServiceService {
	return &MedicalServiceService{repo: repo, docRepo: docRepo}
}

// ServiceInput carries fields for creating or updating a medical service.
type ServiceInput struct {
	DirectionID     int64
	Name            string
	Description     *string
	DurationMinutes int
	Price           *int64 // kopecks
}

func (s *MedicalServiceService) List(ctx context.Context, doctorID int64) ([]model.Service, error) {
	return s.repo.ListByDoctor(ctx, doctorID)
}

func (s *MedicalServiceService) GetByID(ctx context.Context, serviceID int64) (*model.Service, error) {
	return s.repo.GetByID(ctx, serviceID)
}

// Create validates that direction_id belongs to the doctor's assigned directions
// before inserting the service.
func (s *MedicalServiceService) Create(ctx context.Context, doctorID int64, input ServiceInput) (*model.Service, error) {
	dw, err := s.docRepo.GetByID(ctx, doctorID)
	if err != nil {
		return nil, err
	}
	if !doctorHasDirection(dw.Directions, input.DirectionID) {
		return nil, apperrors.ErrDirectionMismatch
	}
	return s.repo.Create(ctx, repository.CreateServiceInput{
		DoctorID:        doctorID,
		DirectionID:     input.DirectionID,
		Name:            input.Name,
		Description:     input.Description,
		DurationMinutes: input.DurationMinutes,
		Price:           input.Price,
	})
}

// Update validates direction ownership before applying changes.
func (s *MedicalServiceService) Update(ctx context.Context, doctorID, serviceID int64, input ServiceInput) (*model.Service, error) {
	dw, err := s.docRepo.GetByID(ctx, doctorID)
	if err != nil {
		return nil, err
	}
	if !doctorHasDirection(dw.Directions, input.DirectionID) {
		return nil, apperrors.ErrDirectionMismatch
	}
	return s.repo.Update(ctx, serviceID, repository.UpdateServiceInput{
		DirectionID:     input.DirectionID,
		Name:            input.Name,
		Description:     input.Description,
		DurationMinutes: input.DurationMinutes,
		Price:           input.Price,
	})
}

func (s *MedicalServiceService) Delete(ctx context.Context, serviceID int64) error {
	return s.repo.SoftDelete(ctx, serviceID)
}

func doctorHasDirection(directions []model.Direction, directionID int64) bool {
	for _, d := range directions {
		if d.ID == directionID {
			return true
		}
	}
	return false
}
