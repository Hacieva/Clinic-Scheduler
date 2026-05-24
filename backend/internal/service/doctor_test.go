package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// mockDoctorRepo implements repository.DoctorRepository for service-layer tests.
type mockDoctorRepo struct {
	doctors    []model.DoctorWithDirections
	doctor     *model.DoctorWithDirections
	doctorRow  *model.Doctor
	err        error
}

func (m *mockDoctorRepo) List(_ context.Context, _ repository.DoctorFilter) ([]model.DoctorWithDirections, error) {
	return m.doctors, m.err
}

func (m *mockDoctorRepo) GetByID(_ context.Context, _ int64) (*model.DoctorWithDirections, error) {
	return m.doctor, m.err
}

func (m *mockDoctorRepo) Create(_ context.Context, input repository.CreateDoctorInput) (*model.Doctor, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Doctor{
		ID: 1, FirstName: input.FirstName, LastName: input.LastName,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockDoctorRepo) Update(_ context.Context, id int64, input repository.UpdateDoctorInput) (*model.Doctor, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Doctor{
		ID: id, FirstName: input.FirstName, LastName: input.LastName,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockDoctorRepo) SoftDelete(_ context.Context, _ int64) error {
	return m.err
}

func (m *mockDoctorRepo) CreateAccount(_ context.Context, _ int64, _ string, _ string) (*model.Doctor, error) {
	return m.doctorRow, m.err
}

func (m *mockDoctorRepo) CreateWithAccount(_ context.Context, input repository.CreateDoctorInput, _ string, _ string) (*model.Doctor, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Doctor{
		ID: 1, FirstName: input.FirstName, LastName: input.LastName,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockDoctorRepo) SetDirections(_ context.Context, _ int64, _ []int64) error {
	return m.err
}

func (m *mockDoctorRepo) GetDoctorIDByUserID(_ context.Context, _ int64) (int64, error) {
	return 0, m.err
}

func sampleDoctorWithDirections() *model.DoctorWithDirections {
	return &model.DoctorWithDirections{
		Doctor: model.Doctor{
			ID: 1, FirstName: "John", LastName: "Smith",
			IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		Directions: []model.Direction{},
	}
}

// — List —

func TestDoctorList_Success(t *testing.T) {
	dw := sampleDoctorWithDirections()
	svc := NewDoctorService(&mockDoctorRepo{doctors: []model.DoctorWithDirections{*dw}}, &mockDirectionRepo{})

	result, err := svc.List(context.Background(), repository.DoctorFilter{})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "John", result[0].FirstName)
}

func TestDoctorList_Empty(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{doctors: []model.DoctorWithDirections{}}, &mockDirectionRepo{})

	result, err := svc.List(context.Background(), repository.DoctorFilter{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

// — GetByID —

func TestDoctorGetByID_Success(t *testing.T) {
	dw := sampleDoctorWithDirections()
	svc := NewDoctorService(&mockDoctorRepo{doctor: dw}, &mockDirectionRepo{})

	result, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
}

func TestDoctorGetByID_NotFound(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	result, err := svc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

// — Create —

func TestDoctorCreate_Success(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{}, &mockDirectionRepo{})

	result, err := svc.Create(context.Background(), DoctorInput{FirstName: "John", LastName: "Smith"})
	require.NoError(t, err)
	assert.Equal(t, "John", result.FirstName)
	assert.Equal(t, "Smith", result.LastName)
}

// — Update —

func TestDoctorUpdate_Success(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{}, &mockDirectionRepo{})

	result, err := svc.Update(context.Background(), 1, DoctorInput{FirstName: "Jane", LastName: "Doe"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
	assert.Equal(t, "Jane", result.FirstName)
}

func TestDoctorUpdate_NotFound(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	result, err := svc.Update(context.Background(), 999, DoctorInput{FirstName: "X", LastName: "Y"})
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

// — Delete —

func TestDoctorDelete_Success(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{}, &mockDirectionRepo{})

	err := svc.Delete(context.Background(), 1)
	require.NoError(t, err)
}

func TestDoctorDelete_NotFound(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	err := svc.Delete(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// — CreateAccount —

func TestDoctorCreateAccount_Success(t *testing.T) {
	uid := int64(10)
	linked := &model.Doctor{
		ID: 1, UserID: &uid, FirstName: "John", LastName: "Smith",
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	svc := NewDoctorService(&mockDoctorRepo{doctorRow: linked}, &mockDirectionRepo{})

	result, err := svc.CreateAccount(context.Background(), 1, "dr@clinic.local", "ValidPass1!")
	require.NoError(t, err)
	require.NotNil(t, result.UserID)
	assert.Equal(t, int64(10), *result.UserID)
}

func TestDoctorCreateAccount_WeakPassword(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{}, &mockDirectionRepo{})

	result, err := svc.CreateAccount(context.Background(), 1, "dr@clinic.local", "short")
	assert.ErrorIs(t, err, auth.ErrWeakPassword)
	assert.Nil(t, result)
}

func TestDoctorCreateAccount_AlreadyHasAccount(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{err: apperrors.ErrAccountExists}, &mockDirectionRepo{})

	result, err := svc.CreateAccount(context.Background(), 1, "dr@clinic.local", "ValidPass1!")
	assert.ErrorIs(t, err, apperrors.ErrAccountExists)
	assert.Nil(t, result)
}

func TestDoctorCreateAccount_DoctorNotFound(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	result, err := svc.CreateAccount(context.Background(), 999, "dr@clinic.local", "ValidPass1!")
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestDoctorCreateAccount_EmailTaken(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{err: apperrors.ErrConflict}, &mockDirectionRepo{})

	result, err := svc.CreateAccount(context.Background(), 1, "taken@clinic.local", "ValidPass1!")
	assert.ErrorIs(t, err, apperrors.ErrConflict)
	assert.Nil(t, result)
}

// — SetDirections —

func TestDoctorSetDirections_Success(t *testing.T) {
	dw := sampleDoctorWithDirections()
	dir := sampleDirection()
	svc := NewDoctorService(
		&mockDoctorRepo{doctor: dw},
		&mockDirectionRepo{direction: &dir},
	)

	err := svc.SetDirections(context.Background(), 1, []int64{1})
	require.NoError(t, err)
}

func TestDoctorSetDirections_EmptyList(t *testing.T) {
	dw := sampleDoctorWithDirections()
	svc := NewDoctorService(&mockDoctorRepo{doctor: dw}, &mockDirectionRepo{})

	err := svc.SetDirections(context.Background(), 1, []int64{})
	require.NoError(t, err)
}

func TestDoctorSetDirections_DoctorNotFound(t *testing.T) {
	svc := NewDoctorService(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	err := svc.SetDirections(context.Background(), 999, []int64{1})
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestDoctorSetDirections_DirectionNotFound(t *testing.T) {
	dw := sampleDoctorWithDirections()
	svc := NewDoctorService(
		&mockDoctorRepo{doctor: dw},
		&mockDirectionRepo{err: apperrors.ErrNotFound},
	)

	err := svc.SetDirections(context.Background(), 1, []int64{999})
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}
