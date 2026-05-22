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

// mockBranchRepo implements repository.BranchRepository.
// hasActiveErr is independent so Deactivate and HasActiveDoctors can return
// different errors in the same test case.
type mockBranchRepo struct {
	branches     []model.Branch
	branch       *model.Branch
	hasActiveDocs bool
	hasActiveErr  error
	err           error
}

func (m *mockBranchRepo) List(_ context.Context) ([]model.Branch, error) {
	return m.branches, m.err
}

func (m *mockBranchRepo) GetByID(_ context.Context, _ int64) (*model.Branch, error) {
	return m.branch, m.err
}

func (m *mockBranchRepo) Create(_ context.Context, input repository.CreateBranchInput) (*model.Branch, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Branch{
		ID: 1, Name: input.Name, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockBranchRepo) Update(_ context.Context, id int64, input repository.UpdateBranchInput) (*model.Branch, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Branch{
		ID: id, Name: input.Name, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockBranchRepo) Deactivate(_ context.Context, _ int64) error {
	return m.err
}

func (m *mockBranchRepo) HasActiveDoctors(_ context.Context, _ int64) (bool, error) {
	return m.hasActiveDocs, m.hasActiveErr
}

func sampleBranch() *model.Branch {
	name := "Главный филиал"
	return &model.Branch{
		ID: 1, Name: name, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// — List —

func TestBranchList_Success(t *testing.T) {
	b := sampleBranch()
	svc := NewBranchService(&mockBranchRepo{branches: []model.Branch{*b}})

	result, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Главный филиал", result[0].Name)
}

func TestBranchList_Empty(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{branches: []model.Branch{}})

	result, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result)
}

// — GetByID —

func TestBranchGetByID_Success(t *testing.T) {
	b := sampleBranch()
	svc := NewBranchService(&mockBranchRepo{branch: b})

	result, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
}

func TestBranchGetByID_NotFound(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{err: apperrors.ErrNotFound})

	result, err := svc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

// — Create —

func TestBranchCreate_Success(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{})

	result, err := svc.Create(context.Background(), BranchInput{Name: "Филиал №2"})
	require.NoError(t, err)
	assert.Equal(t, "Филиал №2", result.Name)
}

func TestBranchCreate_TrimsName(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{})

	result, err := svc.Create(context.Background(), BranchInput{Name: "  Центр  "})
	require.NoError(t, err)
	assert.Equal(t, "Центр", result.Name)
}

func TestBranchCreate_EmptyName(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{})

	result, err := svc.Create(context.Background(), BranchInput{Name: ""})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestBranchCreate_WhitespaceName(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{})

	result, err := svc.Create(context.Background(), BranchInput{Name: "   "})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Nil(t, result)
}

// — Update —

func TestBranchUpdate_Success(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{})

	result, err := svc.Update(context.Background(), 1, BranchInput{Name: "Новое название"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
	assert.Equal(t, "Новое название", result.Name)
}

func TestBranchUpdate_EmptyName(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{})

	result, err := svc.Update(context.Background(), 1, BranchInput{Name: ""})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestBranchUpdate_NotFound(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{err: apperrors.ErrNotFound})

	result, err := svc.Update(context.Background(), 999, BranchInput{Name: "X"})
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

// — Deactivate —

func TestBranchDeactivate_Success(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{hasActiveDocs: false})

	err := svc.Deactivate(context.Background(), 1)
	require.NoError(t, err)
}

func TestBranchDeactivate_HasActiveDoctors(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{hasActiveDocs: true})

	err := svc.Deactivate(context.Background(), 1)
	assert.ErrorIs(t, err, apperrors.ErrBranchHasActiveDoctors)
}

// HasActiveDoctors returns no error but Deactivate returns ErrNotFound (branch gone).
func TestBranchDeactivate_NotFound(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{hasActiveDocs: false, err: apperrors.ErrNotFound})

	err := svc.Deactivate(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestBranchDeactivate_HasActiveDoctorsCheckError(t *testing.T) {
	svc := NewBranchService(&mockBranchRepo{hasActiveErr: apperrors.ErrNotFound})

	err := svc.Deactivate(context.Background(), 1)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}
