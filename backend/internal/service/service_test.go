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

// mockServiceRepo implements repository.ServiceRepository for service-layer tests.
type mockServiceRepo struct {
	services []model.Service
	svc      *model.Service
	err      error
}

func (m *mockServiceRepo) ListByDoctor(_ context.Context, _ int64) ([]model.Service, error) {
	return m.services, m.err
}

func (m *mockServiceRepo) GetByID(_ context.Context, _ int64) (*model.Service, error) {
	return m.svc, m.err
}

func (m *mockServiceRepo) Create(_ context.Context, input repository.CreateServiceInput) (*model.Service, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Service{
		ID:              1,
		DoctorID:        input.DoctorID,
		DirectionID:     input.DirectionID,
		Name:            input.Name,
		Description:     input.Description,
		DurationMinutes: input.DurationMinutes,
		Price:           input.Price,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

func (m *mockServiceRepo) Update(_ context.Context, id int64, input repository.UpdateServiceInput) (*model.Service, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Service{
		ID:              id,
		DirectionID:     input.DirectionID,
		Name:            input.Name,
		Description:     input.Description,
		DurationMinutes: input.DurationMinutes,
		Price:           input.Price,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

func (m *mockServiceRepo) SoftDelete(_ context.Context, _ int64) error {
	return m.err
}

func (m *mockServiceRepo) GetDurationMinutes(_ context.Context, _ int64) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 30, nil
}

// sampleDoctorWithDir returns a doctor that has direction ID=1 assigned.
func sampleDoctorWithDir() *model.DoctorWithDirections {
	return &model.DoctorWithDirections{
		Doctor: model.Doctor{
			ID: 1, FirstName: "John", LastName: "Smith",
			IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		Directions: []model.Direction{
			{ID: 1, Name: "Cardiology", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
	}
}

func sampleSvcInput() ServiceInput {
	return ServiceInput{DirectionID: 1, Name: "Consultation", DurationMinutes: 30}
}

// — List —

func TestMedicalServiceList_Success(t *testing.T) {
	price := int64(150000)
	svcs := []model.Service{{
		ID: 1, DoctorID: 1, DirectionID: 1, Name: "Consultation",
		DurationMinutes: 30, Price: &price, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}
	svc := NewMedicalServiceService(&mockServiceRepo{services: svcs}, &mockDoctorRepo{})

	result, err := svc.List(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Consultation", result[0].Name)
}

func TestMedicalServiceList_Empty(t *testing.T) {
	svc := NewMedicalServiceService(&mockServiceRepo{services: []model.Service{}}, &mockDoctorRepo{})

	result, err := svc.List(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// — GetByID —

func TestMedicalServiceGetByID_Success(t *testing.T) {
	s := &model.Service{
		ID: 1, DoctorID: 1, DirectionID: 1, Name: "Consultation",
		DurationMinutes: 30, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	svc := NewMedicalServiceService(&mockServiceRepo{svc: s}, &mockDoctorRepo{})

	result, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
}

func TestMedicalServiceGetByID_NotFound(t *testing.T) {
	svc := NewMedicalServiceService(&mockServiceRepo{err: apperrors.ErrNotFound}, &mockDoctorRepo{})

	result, err := svc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

// — Create —

func TestMedicalServiceCreate_Success(t *testing.T) {
	dw := sampleDoctorWithDir()
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	result, err := svc.Create(context.Background(), 1, sampleSvcInput())
	require.NoError(t, err)
	assert.Equal(t, "Consultation", result.Name)
	assert.Equal(t, int64(1), result.DoctorID)
}

func TestMedicalServiceCreate_DoctorNotFound(t *testing.T) {
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{err: apperrors.ErrNotFound})

	result, err := svc.Create(context.Background(), 999, sampleSvcInput())
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestMedicalServiceCreate_DirectionMismatch(t *testing.T) {
	// Doctor has direction 2, input requests direction 1.
	dw := &model.DoctorWithDirections{
		Doctor:     model.Doctor{ID: 1, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Directions: []model.Direction{{ID: 2, Name: "Neurology", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}},
	}
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	result, err := svc.Create(context.Background(), 1, sampleSvcInput())
	assert.ErrorIs(t, err, apperrors.ErrDirectionMismatch)
	assert.Nil(t, result)
}

func TestMedicalServiceCreate_PriceAsKopecks(t *testing.T) {
	price := int64(300000) // 3000.00 ₽ = 300 000 kopecks
	dw := sampleDoctorWithDir()
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	result, err := svc.Create(context.Background(), 1, ServiceInput{
		DirectionID:     1,
		Name:            "Premium Consultation",
		DurationMinutes: 60,
		Price:           &price,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Price)
	assert.Equal(t, int64(300000), *result.Price)
}

func TestMedicalServiceCreate_NilPrice(t *testing.T) {
	dw := sampleDoctorWithDir()
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	result, err := svc.Create(context.Background(), 1, ServiceInput{
		DirectionID: 1, Name: "Free Consultation", DurationMinutes: 30,
	})
	require.NoError(t, err)
	assert.Nil(t, result.Price)
}

// — Update —

func TestMedicalServiceUpdate_Success(t *testing.T) {
	dw := sampleDoctorWithDir()
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	result, err := svc.Update(context.Background(), 1, 1, sampleSvcInput())
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
	assert.Equal(t, "Consultation", result.Name)
}

func TestMedicalServiceUpdate_DoctorNotFound(t *testing.T) {
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{err: apperrors.ErrNotFound})

	result, err := svc.Update(context.Background(), 999, 1, sampleSvcInput())
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestMedicalServiceUpdate_DirectionMismatch(t *testing.T) {
	dw := &model.DoctorWithDirections{
		Doctor:     model.Doctor{ID: 1, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Directions: []model.Direction{{ID: 2, Name: "Neurology", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}},
	}
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	result, err := svc.Update(context.Background(), 1, 1, sampleSvcInput())
	assert.ErrorIs(t, err, apperrors.ErrDirectionMismatch)
	assert.Nil(t, result)
}

func TestMedicalServiceUpdate_ServiceNotFound(t *testing.T) {
	dw := sampleDoctorWithDir()
	svc := NewMedicalServiceService(&mockServiceRepo{err: apperrors.ErrNotFound}, &mockDoctorRepo{doctor: dw})

	result, err := svc.Update(context.Background(), 1, 999, sampleSvcInput())
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

// — Delete —

func TestMedicalServiceDelete_Success(t *testing.T) {
	svc := NewMedicalServiceService(&mockServiceRepo{}, &mockDoctorRepo{})

	err := svc.Delete(context.Background(), 1)
	require.NoError(t, err)
}

func TestMedicalServiceDelete_NotFound(t *testing.T) {
	svc := NewMedicalServiceService(&mockServiceRepo{err: apperrors.ErrNotFound}, &mockDoctorRepo{})

	err := svc.Delete(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}
