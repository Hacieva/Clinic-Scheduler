package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// mockScheduleRepo implements repository.ScheduleRepository for service-layer tests.
type mockScheduleRepo struct {
	workingHours []model.WorkingHours
	exceptions   []model.ScheduleException
	exception    *model.ScheduleException
	err          error
	replaceErr   error
}

func (m *mockScheduleRepo) ListWorkingHours(_ context.Context, _ int64) ([]model.WorkingHours, error) {
	return m.workingHours, m.err
}

func (m *mockScheduleRepo) ReplaceWorkingHours(_ context.Context, _ int64, _ []repository.CreateWorkingHoursInput) error {
	return m.replaceErr
}

func (m *mockScheduleRepo) ListExceptions(_ context.Context, _ int64, _, _ time.Time) ([]model.ScheduleException, error) {
	return m.exceptions, m.err
}

func (m *mockScheduleRepo) CreateException(_ context.Context, _ repository.CreateExceptionInput) (*model.ScheduleException, error) {
	return m.exception, m.err
}

func (m *mockScheduleRepo) UpdateException(_ context.Context, _ int64, _ repository.CreateExceptionInput) (*model.ScheduleException, error) {
	return m.exception, m.err
}

func (m *mockScheduleRepo) DeleteException(_ context.Context, _ int64) error {
	return m.err
}

func (m *mockScheduleRepo) CreateExceptionRange(_ context.Context, _ int64, _, _ time.Time, _ model.ExceptionType) (int, error) {
	return 0, m.err
}

// helpers

func todayTime(h, m int) time.Time {
	return time.Date(2000, 1, 1, h, m, 0, 0, time.UTC)
}

func ptr[T any](v T) *T { return &v }

// — ListWorkingHours —

func TestScheduleListWorkingHours(t *testing.T) {
	wh := model.WorkingHours{ID: 1, DoctorID: 1, DayOfWeek: 1,
		StartTime: todayTime(9, 0), EndTime: todayTime(17, 0), IsActive: true}
	svc := NewScheduleService(&mockScheduleRepo{workingHours: []model.WorkingHours{wh}})

	result, err := svc.ListWorkingHours(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 1, result[0].DayOfWeek)
}

// — ReplaceWorkingHours validation —

func TestReplaceWorkingHours_Valid(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	inputs := []WorkingHoursInput{
		{DayOfWeek: 1, StartTime: todayTime(9, 0), EndTime: todayTime(17, 0)},
		{DayOfWeek: 7, StartTime: todayTime(10, 0), EndTime: todayTime(14, 0)},
	}
	err := svc.ReplaceWorkingHours(context.Background(), 1, inputs)
	assert.NoError(t, err)
}

func TestReplaceWorkingHours_InvalidDay_Zero(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	inputs := []WorkingHoursInput{
		{DayOfWeek: 0, StartTime: todayTime(9, 0), EndTime: todayTime(17, 0)},
	}
	err := svc.ReplaceWorkingHours(context.Background(), 1, inputs)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestReplaceWorkingHours_InvalidDay_Eight(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	inputs := []WorkingHoursInput{
		{DayOfWeek: 8, StartTime: todayTime(9, 0), EndTime: todayTime(17, 0)},
	}
	err := svc.ReplaceWorkingHours(context.Background(), 1, inputs)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestReplaceWorkingHours_StartEqualsEnd(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	t0 := todayTime(9, 0)
	inputs := []WorkingHoursInput{{DayOfWeek: 1, StartTime: t0, EndTime: t0}}
	err := svc.ReplaceWorkingHours(context.Background(), 1, inputs)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestReplaceWorkingHours_StartAfterEnd(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	inputs := []WorkingHoursInput{
		{DayOfWeek: 1, StartTime: todayTime(17, 0), EndTime: todayTime(9, 0)},
	}
	err := svc.ReplaceWorkingHours(context.Background(), 1, inputs)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestReplaceWorkingHours_Empty(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	err := svc.ReplaceWorkingHours(context.Background(), 1, []WorkingHoursInput{})
	assert.NoError(t, err)
}

// — CreateException validation —

func TestCreateException_DayOff_Valid(t *testing.T) {
	ex := &model.ScheduleException{ID: 1, Type: model.ExceptionTypeDayOff}
	svc := NewScheduleService(&mockScheduleRepo{exception: ex})
	inp := ExceptionInput{DoctorID: 1, Date: time.Now(), Type: model.ExceptionTypeDayOff}

	result, err := svc.CreateException(context.Background(), inp)
	require.NoError(t, err)
	assert.Equal(t, model.ExceptionTypeDayOff, result.Type)
}

func TestCreateException_DayOff_WithStartTime(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	t0 := todayTime(9, 0)
	inp := ExceptionInput{DoctorID: 1, Date: time.Now(), Type: model.ExceptionTypeDayOff, StartTime: &t0}

	_, err := svc.CreateException(context.Background(), inp)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestCreateException_DayOff_WithEndTime(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	t0 := todayTime(17, 0)
	inp := ExceptionInput{DoctorID: 1, Date: time.Now(), Type: model.ExceptionTypeDayOff, EndTime: &t0}

	_, err := svc.CreateException(context.Background(), inp)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestCreateException_CustomWorkingHours_Valid(t *testing.T) {
	ex := &model.ScheduleException{ID: 1, Type: model.ExceptionTypeCustomWorkingHours}
	svc := NewScheduleService(&mockScheduleRepo{exception: ex})
	start := todayTime(10, 0)
	end := todayTime(14, 0)
	inp := ExceptionInput{
		DoctorID: 1, Date: time.Now(),
		Type:      model.ExceptionTypeCustomWorkingHours,
		StartTime: &start, EndTime: &end,
	}

	result, err := svc.CreateException(context.Background(), inp)
	require.NoError(t, err)
	assert.Equal(t, model.ExceptionTypeCustomWorkingHours, result.Type)
}

func TestCreateException_CustomWorkingHours_MissingStart(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	end := todayTime(14, 0)
	inp := ExceptionInput{
		DoctorID: 1, Date: time.Now(),
		Type:    model.ExceptionTypeCustomWorkingHours,
		EndTime: &end,
	}
	_, err := svc.CreateException(context.Background(), inp)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestCreateException_CustomWorkingHours_MissingEnd(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	start := todayTime(10, 0)
	inp := ExceptionInput{
		DoctorID:  1, Date: time.Now(),
		Type:      model.ExceptionTypeCustomWorkingHours,
		StartTime: &start,
	}
	_, err := svc.CreateException(context.Background(), inp)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestCreateException_CustomWorkingHours_StartAfterEnd(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	start := todayTime(14, 0)
	end := todayTime(10, 0)
	inp := ExceptionInput{
		DoctorID: 1, Date: time.Now(),
		Type:      model.ExceptionTypeCustomWorkingHours,
		StartTime: &start, EndTime: &end,
	}
	_, err := svc.CreateException(context.Background(), inp)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestCreateException_InvalidType(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	inp := ExceptionInput{DoctorID: 1, Date: time.Now(), Type: model.ExceptionType("unknown")}
	_, err := svc.CreateException(context.Background(), inp)
	assert.ErrorIs(t, err, apperrors.ErrInvalidSchedule)
}

func TestCreateException_Conflict(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{err: apperrors.ErrConflict})
	inp := ExceptionInput{DoctorID: 1, Date: time.Now(), Type: model.ExceptionTypeDayOff}
	_, err := svc.CreateException(context.Background(), inp)
	assert.ErrorIs(t, err, apperrors.ErrConflict)
}

// — UpdateException —

func TestUpdateException_Valid(t *testing.T) {
	ex := &model.ScheduleException{ID: 5, Type: model.ExceptionTypeDayOff}
	svc := NewScheduleService(&mockScheduleRepo{exception: ex})
	inp := ExceptionInput{DoctorID: 1, Date: time.Now(), Type: model.ExceptionTypeDayOff}

	result, err := svc.UpdateException(context.Background(), 5, inp)
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.ID)
}

func TestUpdateException_NotFound(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{err: apperrors.ErrNotFound})
	inp := ExceptionInput{DoctorID: 1, Date: time.Now(), Type: model.ExceptionTypeDayOff}
	_, err := svc.UpdateException(context.Background(), 99, inp)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// — DeleteException —

func TestDeleteException_Valid(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{})
	err := svc.DeleteException(context.Background(), 1)
	assert.NoError(t, err)
}

func TestDeleteException_NotFound(t *testing.T) {
	svc := NewScheduleService(&mockScheduleRepo{err: apperrors.ErrNotFound})
	err := svc.DeleteException(context.Background(), 99)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}
