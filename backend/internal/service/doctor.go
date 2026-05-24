package service

import (
	"context"

	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type DoctorService struct {
	repo    repository.DoctorRepository
	dirRepo repository.DirectionRepository
}

func NewDoctorService(repo repository.DoctorRepository, dirRepo repository.DirectionRepository) *DoctorService {
	return &DoctorService{repo: repo, dirRepo: dirRepo}
}

// DoctorInput carries fields for creating or updating a doctor profile.
type DoctorInput struct {
	FirstName   string
	LastName    string
	MiddleName  *string
	Cabinet     *string
	BranchID    *int64
	Phone       *string
	Description *string
	PhotoURL    *string
}

func (s *DoctorService) List(ctx context.Context, filter repository.DoctorFilter) ([]model.DoctorWithDirections, error) {
	return s.repo.List(ctx, filter)
}

func (s *DoctorService) GetByID(ctx context.Context, id int64) (*model.DoctorWithDirections, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DoctorService) Create(ctx context.Context, input DoctorInput) (*model.Doctor, error) {
	return s.repo.Create(ctx, repository.CreateDoctorInput{
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		MiddleName:  input.MiddleName,
		Cabinet:     input.Cabinet,
		BranchID:    input.BranchID,
		Phone:       input.Phone,
		Description: input.Description,
		PhotoURL:    input.PhotoURL,
	})
}

// CreateWithAccount atomically creates the doctor profile and a linked user account.
func (s *DoctorService) CreateWithAccount(ctx context.Context, input DoctorInput, email, password string) (*model.Doctor, error) {
	if err := auth.ValidatePasswordStrength(password); err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateWithAccount(ctx, repository.CreateDoctorInput{
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		MiddleName:  input.MiddleName,
		Cabinet:     input.Cabinet,
		BranchID:    input.BranchID,
		Phone:       input.Phone,
		Description: input.Description,
		PhotoURL:    input.PhotoURL,
	}, email, hash)
}

func (s *DoctorService) Update(ctx context.Context, id int64, input DoctorInput) (*model.Doctor, error) {
	return s.repo.Update(ctx, id, repository.UpdateDoctorInput{
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		MiddleName:  input.MiddleName,
		Cabinet:     input.Cabinet,
		BranchID:    input.BranchID,
		Phone:       input.Phone,
		Description: input.Description,
		PhotoURL:    input.PhotoURL,
	})
}

func (s *DoctorService) Delete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}

// CreateAccount validates the password, hashes it, then atomically creates a
// user account and links it to the doctor. The doctor's user_id is set;
// the doctor entity itself is not deactivated or removed on soft-delete.
func (s *DoctorService) CreateAccount(ctx context.Context, doctorID int64, email, password string) (*model.Doctor, error) {
	if err := auth.ValidatePasswordStrength(password); err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateAccount(ctx, doctorID, email, hash)
}

// SetDirections validates that the doctor and every direction exist, then
// atomically replaces the doctor's full set of directions.
func (s *DoctorService) SetDirections(ctx context.Context, doctorID int64, directionIDs []int64) error {
	if _, err := s.repo.GetByID(ctx, doctorID); err != nil {
		return err
	}
	for _, id := range directionIDs {
		if _, err := s.dirRepo.GetByID(ctx, id); err != nil {
			return err
		}
	}
	return s.repo.SetDirections(ctx, doctorID, directionIDs)
}
