package service

import (
	"context"
	"strings"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type BranchService struct {
	repo repository.BranchRepository
}

func NewBranchService(repo repository.BranchRepository) *BranchService {
	return &BranchService{repo: repo}
}

type BranchInput struct {
	Name    string
	Address *string
	Phone   *string
}

func (s *BranchService) List(ctx context.Context) ([]model.Branch, error) {
	return s.repo.List(ctx)
}

func (s *BranchService) GetByID(ctx context.Context, id int64) (*model.Branch, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BranchService) Create(ctx context.Context, input BranchInput) (*model.Branch, error) {
	if err := validateBranchInput(input); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, repository.CreateBranchInput{
		Name:    strings.TrimSpace(input.Name),
		Address: input.Address,
		Phone:   input.Phone,
	})
}

func (s *BranchService) Update(ctx context.Context, id int64, input BranchInput) (*model.Branch, error) {
	if err := validateBranchInput(input); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, repository.UpdateBranchInput{
		Name:    strings.TrimSpace(input.Name),
		Address: input.Address,
		Phone:   input.Phone,
	})
}

// Deactivate marks a branch as inactive.
// Returns ErrBranchHasActiveDoctors if any active doctor is still assigned to it.
func (s *BranchService) Deactivate(ctx context.Context, id int64) error {
	has, err := s.repo.HasActiveDoctors(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return apperrors.ErrBranchHasActiveDoctors
	}
	return s.repo.Deactivate(ctx, id)
}

func validateBranchInput(input BranchInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return apperrors.ErrInvalidInput
	}
	return nil
}
