package service

import (
	"context"

	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// DoctorAssignmentService manages which services a doctor is assigned to perform.
// The doctor_services junction table is the authoritative source.
type DoctorAssignmentService struct {
	repo    repository.DoctorServiceRepository
	svcRepo repository.ServiceRepository
	docRepo repository.DoctorRepository
}

func NewDoctorAssignmentService(
	repo repository.DoctorServiceRepository,
	svcRepo repository.ServiceRepository,
	docRepo repository.DoctorRepository,
) *DoctorAssignmentService {
	return &DoctorAssignmentService{repo: repo, svcRepo: svcRepo, docRepo: docRepo}
}

func (s *DoctorAssignmentService) ListForDoctor(ctx context.Context, doctorID int64) ([]model.Service, error) {
	return s.repo.ListAssignedToDoctor(ctx, doctorID)
}

func (s *DoctorAssignmentService) Assign(ctx context.Context, doctorID, serviceID int64) error {
	if _, err := s.docRepo.GetByID(ctx, doctorID); err != nil {
		return err
	}
	if _, err := s.svcRepo.GetByID(ctx, serviceID); err != nil {
		return err
	}
	return s.repo.Assign(ctx, doctorID, serviceID)
}

func (s *DoctorAssignmentService) Unassign(ctx context.Context, doctorID, serviceID int64) error {
	return s.repo.Unassign(ctx, doctorID, serviceID)
}

// BulkSet replaces all service assignments for a doctor with the given list.
func (s *DoctorAssignmentService) BulkSet(ctx context.Context, doctorID int64, serviceIDs []int64) error {
	if _, err := s.docRepo.GetByID(ctx, doctorID); err != nil {
		return err
	}
	return s.repo.BulkReplace(ctx, doctorID, serviceIDs)
}
