package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

type mockDirectionRepo struct {
	directions []model.Direction
	direction  *model.Direction
	err        error
}

func (m *mockDirectionRepo) List(_ context.Context) ([]model.Direction, error) {
	return m.directions, m.err
}

func (m *mockDirectionRepo) GetByID(_ context.Context, _ int64) (*model.Direction, error) {
	return m.direction, m.err
}

func (m *mockDirectionRepo) Create(_ context.Context, name string, description *string) (*model.Direction, error) {
	if m.err != nil {
		return nil, m.err
	}
	d := &model.Direction{
		ID:          1,
		Name:        name,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return d, nil
}

func (m *mockDirectionRepo) Update(_ context.Context, id int64, name string, description *string) (*model.Direction, error) {
	if m.err != nil {
		return nil, m.err
	}
	d := &model.Direction{
		ID:          id,
		Name:        name,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return d, nil
}

func (m *mockDirectionRepo) SoftDelete(_ context.Context, _ int64) error {
	return m.err
}

func sampleDirection() model.Direction {
	desc := "treats hearts"
	return model.Direction{
		ID:          1,
		Name:        "Cardiology",
		Description: &desc,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestDirectionList_Success(t *testing.T) {
	dirs := []model.Direction{sampleDirection()}
	svc := NewDirectionService(&mockDirectionRepo{directions: dirs})

	result, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Cardiology", result[0].Name)
}

func TestDirectionList_Empty(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{directions: []model.Direction{}})

	result, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDirectionGetByID_Success(t *testing.T) {
	d := sampleDirection()
	svc := NewDirectionService(&mockDirectionRepo{direction: &d})

	result, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
	assert.Equal(t, "Cardiology", result.Name)
}

func TestDirectionGetByID_NotFound(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{err: apperrors.ErrNotFound})

	result, err := svc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestDirectionCreate_Success(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{})

	desc := "treats hearts"
	result, err := svc.Create(context.Background(), "Cardiology", &desc)
	require.NoError(t, err)
	assert.Equal(t, "Cardiology", result.Name)
	assert.Equal(t, &desc, result.Description)
	assert.True(t, result.IsActive)
}

func TestDirectionCreate_NilDescription(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{})

	result, err := svc.Create(context.Background(), "General", nil)
	require.NoError(t, err)
	assert.Equal(t, "General", result.Name)
	assert.Nil(t, result.Description)
}

func TestDirectionUpdate_Success(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{})

	desc := "updated"
	result, err := svc.Update(context.Background(), 1, "Updated", &desc)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
	assert.Equal(t, "Updated", result.Name)
}

func TestDirectionUpdate_NotFound(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{err: apperrors.ErrNotFound})

	result, err := svc.Update(context.Background(), 999, "X", nil)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestDirectionDelete_Success(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{})

	err := svc.Delete(context.Background(), 1)
	require.NoError(t, err)
}

func TestDirectionDelete_NotFound(t *testing.T) {
	svc := NewDirectionService(&mockDirectionRepo{err: apperrors.ErrNotFound})

	err := svc.Delete(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}
