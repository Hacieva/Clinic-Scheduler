package service

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type PatientService struct {
	repo repository.PatientRepository
}

func NewPatientService(repo repository.PatientRepository) *PatientService {
	return &PatientService{repo: repo}
}

func (s *PatientService) List(ctx context.Context, filter repository.PatientFilter) ([]model.Patient, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.repo.List(ctx, filter)
}

func (s *PatientService) GetByID(ctx context.Context, id int64) (*model.Patient, error) {
	return s.repo.GetByID(ctx, id)
}

// Create upserts by phone: if a patient with the same phone already exists, the
// existing record is returned unchanged. Otherwise a new patient is inserted.
func (s *PatientService) Create(ctx context.Context, input repository.CreatePatientInput) (*model.Patient, error) {
	if strings.TrimSpace(input.FullName) == "" {
		return nil, apperrors.ErrInvalidInput
	}
	if strings.TrimSpace(input.Phone) == "" {
		return nil, apperrors.ErrInvalidInput
	}
	if input.Source == "" {
		input.Source = "admin_panel"
	}

	existing, err := s.repo.GetByPhone(ctx, input.Phone)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}
	return s.repo.Create(ctx, input)
}

func (s *PatientService) Update(ctx context.Context, id int64, input repository.UpdatePatientInput) (*model.Patient, error) {
	if input.FullName != nil && strings.TrimSpace(*input.FullName) == "" {
		return nil, apperrors.ErrInvalidInput
	}
	if input.Phone != nil && strings.TrimSpace(*input.Phone) == "" {
		return nil, apperrors.ErrInvalidInput
	}
	return s.repo.Update(ctx, id, input)
}
