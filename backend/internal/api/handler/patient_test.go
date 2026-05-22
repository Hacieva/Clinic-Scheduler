package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hacieva/clinic-scheduler/backend/internal/api/middleware"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

// mockPatientSvcRepo implements repository.PatientRepository for handler tests.
type mockPatientSvcRepo struct {
	patient    *model.Patient
	patients   []model.Patient
	byPhoneErr error
	err        error
}

func (m *mockPatientSvcRepo) List(_ context.Context, _ repository.PatientFilter) ([]model.Patient, error) {
	return m.patients, m.err
}
func (m *mockPatientSvcRepo) GetByID(_ context.Context, _ int64) (*model.Patient, error) {
	return m.patient, m.err
}
func (m *mockPatientSvcRepo) GetByPhone(_ context.Context, _ string) (*model.Patient, error) {
	return m.patient, m.byPhoneErr
}
func (m *mockPatientSvcRepo) Create(_ context.Context, _ repository.CreatePatientInput) (*model.Patient, error) {
	return m.patient, m.err
}
func (m *mockPatientSvcRepo) Update(_ context.Context, _ int64, _ repository.UpdatePatientInput) (*model.Patient, error) {
	return m.patient, m.err
}

func samplePatientModel() *model.Patient {
	return &model.Patient{
		ID:        1,
		FullName:  "Иванов Иван",
		Phone:     "+79001234567",
		Source:    "admin_panel",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newPatientRouter(repo *mockPatientSvcRepo) http.Handler {
	svc := service.NewPatientService(repo)
	h := NewPatientHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Authenticate(testSecret))
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("owner", "admin"))
		r.Get("/patients", h.List)
		r.Get("/patients/{id}", h.GetByID)
		r.Post("/patients", h.Create)
		r.Patch("/patients/{id}", h.Update)
	})
	return r
}

// — List —

func TestPatientList_Admin(t *testing.T) {
	repo := &mockPatientSvcRepo{patients: []model.Patient{*samplePatientModel()}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var patients []model.Patient
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &patients))
	assert.Len(t, patients, 1)
}

func TestPatientList_Owner(t *testing.T) {
	repo := &mockPatientSvcRepo{patients: []model.Patient{*samplePatientModel()}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestPatientList_DoctorForbidden(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients", nil)
	req.Header.Set("Authorization", "Bearer "+doctorToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestPatientList_Unauthenticated(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients", nil)

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestPatientList_InvalidLimit(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients?limit=abc", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// — GetByID —

func TestPatientGetByID_Success(t *testing.T) {
	repo := &mockPatientSvcRepo{patient: samplePatientModel()}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients/1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var p model.Patient
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &p))
	assert.Equal(t, int64(1), p.ID)
}

func TestPatientGetByID_NotFound(t *testing.T) {
	repo := &mockPatientSvcRepo{err: apperrors.ErrNotFound}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients/99", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPatientGetByID_InvalidID(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/patients/abc", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// — Create —

func TestPatientCreate_Success(t *testing.T) {
	repo := &mockPatientSvcRepo{
		byPhoneErr: apperrors.ErrNotFound,
		patient:    samplePatientModel(),
	}
	body := `{"full_name":"Иванов Иван","phone":"+79001234567"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	var p model.Patient
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &p))
	assert.Equal(t, int64(1), p.ID)
}

func TestPatientCreate_UpsertReturnsExisting(t *testing.T) {
	existing := samplePatientModel()
	repo := &mockPatientSvcRepo{
		byPhoneErr: nil, // phone found
		patient:    existing,
	}
	body := `{"full_name":"Другое Имя","phone":"+79001234567"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	// 201 with the existing patient
	require.Equal(t, http.StatusCreated, rr.Code)
}

func TestPatientCreate_MissingFullName(t *testing.T) {
	body := `{"phone":"+79001234567"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPatientCreate_MissingPhone(t *testing.T) {
	body := `{"full_name":"Иванов Иван"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPatientCreate_InvalidDateOfBirth(t *testing.T) {
	body := `{"full_name":"Иванов","phone":"+7900","date_of_birth":"not-a-date"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPatientCreate_DoctorForbidden(t *testing.T) {
	body := `{"full_name":"Иванов Иван","phone":"+79001234567"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+doctorToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — Update —

func TestPatientUpdate_Success(t *testing.T) {
	repo := &mockPatientSvcRepo{patient: samplePatientModel()}
	body := `{"full_name":"Новое Имя"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/patients/1", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestPatientUpdate_NotFound(t *testing.T) {
	repo := &mockPatientSvcRepo{err: apperrors.ErrNotFound}
	body := `{"full_name":"Новое Имя"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/patients/99", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPatientUpdate_InvalidDateOfBirth(t *testing.T) {
	body := `{"date_of_birth":"bad-date"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/patients/1", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	newPatientRouter(&mockPatientSvcRepo{}).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPatientUpdate_OwnerAccess(t *testing.T) {
	repo := &mockPatientSvcRepo{patient: samplePatientModel()}
	body := `{"comment":"VIP"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/patients/1", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+ownerToken(t))

	newPatientRouter(repo).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
