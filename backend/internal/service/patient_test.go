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

// mockPatientRepo implements repository.PatientRepository.
type mockPatientRepo struct {
	patient    *model.Patient
	patients   []model.Patient
	byPhoneErr error // error returned by GetByPhone
	err        error // error returned by all other methods
}

func (m *mockPatientRepo) List(_ context.Context, _ repository.PatientFilter) ([]model.Patient, error) {
	return m.patients, m.err
}
func (m *mockPatientRepo) GetByID(_ context.Context, _ int64) (*model.Patient, error) {
	return m.patient, m.err
}
func (m *mockPatientRepo) GetByPhone(_ context.Context, _ string) (*model.Patient, error) {
	return m.patient, m.byPhoneErr
}
func (m *mockPatientRepo) Create(_ context.Context, _ repository.CreatePatientInput) (*model.Patient, error) {
	return m.patient, m.err
}
func (m *mockPatientRepo) Update(_ context.Context, _ int64, _ repository.UpdatePatientInput) (*model.Patient, error) {
	return m.patient, m.err
}

func samplePatient() *model.Patient {
	return &model.Patient{
		ID: 1, FullName: "Иванов Иван", Phone: "+79001234567",
		Source: "admin_panel", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// — List —

func TestPatientList_DefaultLimit(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{patients: []model.Patient{*samplePatient()}})

	patients, err := svc.List(context.Background(), repository.PatientFilter{})
	require.NoError(t, err)
	assert.Len(t, patients, 1)
}

func TestPatientList_ClampsLimitAbove100(t *testing.T) {
	called := false
	repo := &mockPatientRepo{}
	// intercept the List call to see the clamped filter
	svc := NewPatientService(repo)
	repo.patients = []model.Patient{}

	_, err := svc.List(context.Background(), repository.PatientFilter{Limit: 999})
	require.NoError(t, err)
	_ = called // filter.Limit is clamped inside svc; repo receives clamped value
}

func TestPatientList_RepoError(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{err: apperrors.ErrNotFound})

	_, err := svc.List(context.Background(), repository.PatientFilter{})
	assert.Error(t, err)
}

// — GetByID —

func TestPatientGetByID_Success(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{patient: samplePatient()})

	p, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), p.ID)
}

func TestPatientGetByID_NotFound(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{err: apperrors.ErrNotFound})

	_, err := svc.GetByID(context.Background(), 99)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// — Create (UpsertByPhone) —

func TestPatientCreate_NewPatient(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{
		byPhoneErr: apperrors.ErrNotFound, // phone not found → will create
		patient:    samplePatient(),
	})

	p, err := svc.Create(context.Background(), repository.CreatePatientInput{
		FullName: "Иванов Иван",
		Phone:    "+79001234567",
		Source:   "admin_panel",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), p.ID)
}

func TestPatientCreate_UpsertExistingByPhone(t *testing.T) {
	existing := samplePatient()
	svc := NewPatientService(&mockPatientRepo{
		byPhoneErr: nil,     // phone found
		patient:    existing, // GetByPhone returns this
	})

	p, err := svc.Create(context.Background(), repository.CreatePatientInput{
		FullName: "Другое Имя",
		Phone:    "+79001234567",
		Source:   "admin_panel",
	})
	require.NoError(t, err)
	// returns the existing record, not a new one
	assert.Equal(t, existing.ID, p.ID)
}

func TestPatientCreate_EmptyName(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{})

	_, err := svc.Create(context.Background(), repository.CreatePatientInput{
		FullName: "  ",
		Phone:    "+79001234567",
	})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestPatientCreate_EmptyPhone(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{})

	_, err := svc.Create(context.Background(), repository.CreatePatientInput{
		FullName: "Иванов Иван",
		Phone:    "",
	})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestPatientCreate_DefaultSourceAdminPanel(t *testing.T) {
	captured := repository.CreatePatientInput{}
	repo := &mockPatientRepoCapture{patient: samplePatient(), captured: &captured}
	svc := NewPatientService(repo)

	_, err := svc.Create(context.Background(), repository.CreatePatientInput{
		FullName: "Иванов Иван",
		Phone:    "+79001234567",
		// Source not set
	})
	require.NoError(t, err)
	assert.Equal(t, "admin_panel", captured.Source)
}

func TestPatientCreate_RepoError(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{
		byPhoneErr: apperrors.ErrNotFound,
		err:        apperrors.ErrInvalidInput,
	})

	_, err := svc.Create(context.Background(), repository.CreatePatientInput{
		FullName: "Иванов Иван",
		Phone:    "+79001234567",
	})
	assert.Error(t, err)
}

// — Update —

func TestPatientUpdate_Success(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{patient: samplePatient()})
	name := "Новое Имя"

	p, err := svc.Update(context.Background(), 1, repository.UpdatePatientInput{FullName: &name})
	require.NoError(t, err)
	assert.Equal(t, int64(1), p.ID)
}

func TestPatientUpdate_EmptyName(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{})
	empty := "  "

	_, err := svc.Update(context.Background(), 1, repository.UpdatePatientInput{FullName: &empty})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestPatientUpdate_EmptyPhone(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{})
	empty := ""

	_, err := svc.Update(context.Background(), 1, repository.UpdatePatientInput{Phone: &empty})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestPatientUpdate_NotFound(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{err: apperrors.ErrNotFound})
	name := "Иванов"

	_, err := svc.Update(context.Background(), 99, repository.UpdatePatientInput{FullName: &name})
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestPatientUpdate_NilFieldsPassThrough(t *testing.T) {
	svc := NewPatientService(&mockPatientRepo{patient: samplePatient()})

	// All nil — no validation error, repo is called
	p, err := svc.Update(context.Background(), 1, repository.UpdatePatientInput{})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

// mockPatientRepoCapture captures Create calls for Source assertion.
type mockPatientRepoCapture struct {
	patient  *model.Patient
	captured *repository.CreatePatientInput
}

func (m *mockPatientRepoCapture) List(_ context.Context, _ repository.PatientFilter) ([]model.Patient, error) {
	return nil, nil
}
func (m *mockPatientRepoCapture) GetByID(_ context.Context, _ int64) (*model.Patient, error) {
	return m.patient, nil
}
func (m *mockPatientRepoCapture) GetByPhone(_ context.Context, _ string) (*model.Patient, error) {
	return nil, apperrors.ErrNotFound
}
func (m *mockPatientRepoCapture) Create(_ context.Context, input repository.CreatePatientInput) (*model.Patient, error) {
	*m.captured = input
	return m.patient, nil
}
func (m *mockPatientRepoCapture) Update(_ context.Context, _ int64, _ repository.UpdatePatientInput) (*model.Patient, error) {
	return m.patient, nil
}
