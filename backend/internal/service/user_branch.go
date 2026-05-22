package service

import (
	"context"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type UserBranchService struct {
	repo       repository.UserBranchRepository
	branchRepo repository.BranchRepository
}

func NewUserBranchService(repo repository.UserBranchRepository, branchRepo repository.BranchRepository) *UserBranchService {
	return &UserBranchService{repo: repo, branchRepo: branchRepo}
}

// GetBranchIDs returns the branch IDs the user can access.
// Owner: all active branch IDs from the branches table.
// Admin/Doctor: their assigned IDs from user_branches.
func (s *UserBranchService) GetBranchIDs(ctx context.Context, userID int64, role model.UserRole) ([]int64, error) {
	if role == model.RoleOwner {
		branches, err := s.branchRepo.List(ctx)
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(branches))
		for _, b := range branches {
			if b.IsActive {
				ids = append(ids, b.ID)
			}
		}
		return ids, nil
	}
	return s.repo.GetBranchIDs(ctx, userID)
}

// SetBranchIDs replaces branch assignments for a non-owner user.
// Returns ErrInvalidInput when:
//   - role is owner (automatic full access, no assignment needed)
//   - branchIDs is empty or nil for admin/doctor (would lock the user out)
func (s *UserBranchService) SetBranchIDs(ctx context.Context, userID int64, role model.UserRole, branchIDs []int64) error {
	if role == model.RoleOwner {
		return apperrors.ErrInvalidInput
	}
	if len(branchIDs) == 0 {
		return apperrors.ErrInvalidInput
	}
	return s.repo.SetBranchIDs(ctx, userID, branchIDs)
}

// HasAccess reports whether the user can access the given branch.
// Owner: always true, no DB query.
// Admin/Doctor: checks user_branches table.
func (s *UserBranchService) HasAccess(ctx context.Context, userID int64, role model.UserRole, branchID int64) (bool, error) {
	if role == model.RoleOwner {
		return true, nil
	}
	return s.repo.HasAccess(ctx, userID, branchID)
}
