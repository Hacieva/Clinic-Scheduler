package service

import (
	"context"
	"time"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type ScheduleService struct {
	repo repository.ScheduleRepository
}

func NewScheduleService(repo repository.ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

type WorkingHoursInput struct {
	DayOfWeek int
	StartTime time.Time
	EndTime   time.Time
}

type ExceptionInput struct {
	DoctorID  int64
	Date      time.Time
	Type      model.ExceptionType
	StartTime *time.Time
	EndTime   *time.Time
	Comment   *string
}

func (s *ScheduleService) ListWorkingHours(ctx context.Context, doctorID int64) ([]model.WorkingHours, error) {
	return s.repo.ListWorkingHours(ctx, doctorID)
}

func (s *ScheduleService) ReplaceWorkingHours(ctx context.Context, doctorID int64, inputs []WorkingHoursInput) error {
	repoInputs := make([]repository.CreateWorkingHoursInput, 0, len(inputs))
	for _, inp := range inputs {
		if err := validateWorkingHoursInput(inp); err != nil {
			return err
		}
		repoInputs = append(repoInputs, repository.CreateWorkingHoursInput{
			DayOfWeek: inp.DayOfWeek,
			StartTime: inp.StartTime,
			EndTime:   inp.EndTime,
		})
	}
	return s.repo.ReplaceWorkingHours(ctx, doctorID, repoInputs)
}

func (s *ScheduleService) ListExceptions(ctx context.Context, doctorID int64, from, to time.Time) ([]model.ScheduleException, error) {
	return s.repo.ListExceptions(ctx, doctorID, from, to)
}

func (s *ScheduleService) CreateException(ctx context.Context, input ExceptionInput) (*model.ScheduleException, error) {
	if err := validateExceptionInput(input); err != nil {
		return nil, err
	}
	return s.repo.CreateException(ctx, repository.CreateExceptionInput{
		DoctorID:  input.DoctorID,
		Date:      input.Date,
		Type:      input.Type,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
		Comment:   input.Comment,
	})
}

func (s *ScheduleService) UpdateException(ctx context.Context, id int64, input ExceptionInput) (*model.ScheduleException, error) {
	if err := validateExceptionInput(input); err != nil {
		return nil, err
	}
	return s.repo.UpdateException(ctx, id, repository.CreateExceptionInput{
		DoctorID:  input.DoctorID,
		Date:      input.Date,
		Type:      input.Type,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
		Comment:   input.Comment,
	})
}

func (s *ScheduleService) DeleteException(ctx context.Context, id int64) error {
	return s.repo.DeleteException(ctx, id)
}

func validateWorkingHoursInput(inp WorkingHoursInput) error {
	if inp.DayOfWeek < 1 || inp.DayOfWeek > 7 {
		return apperrors.ErrInvalidSchedule
	}
	if !inp.StartTime.Before(inp.EndTime) {
		return apperrors.ErrInvalidSchedule
	}
	return nil
}

func validateExceptionInput(inp ExceptionInput) error {
	switch inp.Type {
	case model.ExceptionTypeDayOff:
		if inp.StartTime != nil || inp.EndTime != nil {
			return apperrors.ErrInvalidSchedule
		}
	case model.ExceptionTypeCustomWorkingHours:
		if inp.StartTime == nil || inp.EndTime == nil {
			return apperrors.ErrInvalidSchedule
		}
		if !inp.StartTime.Before(*inp.EndTime) {
			return apperrors.ErrInvalidSchedule
		}
	default:
		return apperrors.ErrInvalidSchedule
	}
	return nil
}
