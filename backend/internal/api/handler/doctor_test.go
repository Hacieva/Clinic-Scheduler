package handler

import (
	"context"
	"encoding/json"
	"net/http"
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

// mockDoctorRepo implements repository.DoctorRepository for handler-layer tests.
type mockDoctorRepo struct {
	doctors    []model.DoctorWithDirections
	doctor     *model.DoctorWithDirections
	doctorRow  *model.Doctor
	err        error
}

func (m *mockDoctorRepo) List(_ context.Context, _ *int64) ([]model.DoctorWithDirections, error) {
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

func (m *mockDoctorRepo) SetDirections(_ context.Context, _ int64, _ []int64) error {
	return m.err
}

func (m *mockDoctorRepo) GetDoctorIDByUserID(_ context.Context, _ int64) (int64, error) {
	return 0, m.err
}

func newDoctorRouter(docRepo *mockDoctorRepo, dirRepo *mockDirectionRepo) http.Handler {
	svc := service.NewDoctorService(docRepo, dirRepo)
	h := NewDoctorHandler(svc)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Get("/api/v1/doctors", h.List)
		r.Get("/api/v1/doctors/{id}", h.GetByID)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/api/v1/doctors", h.Create)
			r.Patch("/api/v1/doctors/{id}", h.Update)
			r.Delete("/api/v1/doctors/{id}", h.Delete)
			r.Post("/api/v1/doctors/{id}/account", h.CreateAccount)
			r.Put("/api/v1/doctors/{id}/directions", h.SetDirections)
		})
	})
	return r
}

func sampleDoctorRow() *model.Doctor {
	return &model.Doctor{
		ID: 1, FirstName: "John", LastName: "Smith",
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func sampleDW() model.DoctorWithDirections {
	return model.DoctorWithDirections{
		Doctor:     *sampleDoctorRow(),
		Directions: []model.Direction{},
	}
}

// — List —

func TestDoctorList_Success(t *testing.T) {
	router := newDoctorRouter(
		&mockDoctorRepo{doctors: []model.DoctorWithDirections{sampleDW()}},
		&mockDirectionRepo{},
	)

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.DoctorWithDirections
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestDoctorList_DoctorAllowed(t *testing.T) {
	router := newDoctorRouter(
		&mockDoctorRepo{doctors: []model.DoctorWithDirections{}},
		&mockDirectionRepo{},
	)

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors", "", doctorToken(t))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDoctorList_NoToken(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors", "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — GetByID —

func TestDoctorGetByID_Success(t *testing.T) {
	dw := sampleDW()
	router := newDoctorRouter(&mockDoctorRepo{doctor: &dw}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.DoctorWithDirections
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.ID)
}

func TestDoctorGetByID_NotFound(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/999", "", adminToken(t))

	require.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "doctor not found", resp["error"])
}

func TestDoctorGetByID_InvalidID(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/abc", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// — Create —

func TestDoctorCreate_Success(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors",
		`{"first_name":"John","last_name":"Smith"}`, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Doctor
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "John", resp.FirstName)
}

func TestDoctorCreate_MissingName(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	cases := []struct{ body string }{
		{`{"first_name":"John"}`},
		{`{"last_name":"Smith"}`},
		{`{}`},
	}
	for _, tc := range cases {
		rr := bearerReq(router, http.MethodPost, "/api/v1/doctors", tc.body, adminToken(t))
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	}
}

func TestDoctorCreate_DoctorForbidden(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors",
		`{"first_name":"John","last_name":"Smith"}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestDoctorCreate_NoToken(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors",
		`{"first_name":"John","last_name":"Smith"}`, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — Update (PATCH) —

func TestDoctorUpdate_Success(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPatch, "/api/v1/doctors/1",
		`{"first_name":"Jane","last_name":"Doe"}`, adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.Doctor
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Jane", resp.FirstName)
}

func TestDoctorUpdate_NotFound(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPatch, "/api/v1/doctors/999",
		`{"first_name":"X","last_name":"Y"}`, adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDoctorUpdate_DoctorForbidden(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPatch, "/api/v1/doctors/1",
		`{"first_name":"X","last_name":"Y"}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — Delete —

func TestDoctorDelete_Success(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1", "", adminToken(t))

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDoctorDelete_NotFound(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/999", "", adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDoctorDelete_DoctorForbidden(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1", "", doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — CreateAccount —

func TestDoctorCreateAccount_Success(t *testing.T) {
	uid := int64(10)
	linked := &model.Doctor{
		ID: 1, UserID: &uid, FirstName: "John", LastName: "Smith",
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	router := newDoctorRouter(&mockDoctorRepo{doctorRow: linked}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/account",
		`{"email":"dr@clinic.local","password":"ValidPass1!"}`, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Doctor
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotNil(t, resp.UserID)
	assert.Equal(t, int64(10), *resp.UserID)
}

func TestDoctorCreateAccount_AlreadyHasAccount(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{err: apperrors.ErrAccountExists}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/account",
		`{"email":"dr@clinic.local","password":"ValidPass1!"}`, adminToken(t))

	require.Equal(t, http.StatusConflict, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "doctor already has an account", resp["error"])
}

func TestDoctorCreateAccount_EmailTaken(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{err: apperrors.ErrConflict}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/account",
		`{"email":"taken@clinic.local","password":"ValidPass1!"}`, adminToken(t))

	require.Equal(t, http.StatusConflict, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "email already taken", resp["error"])
}

func TestDoctorCreateAccount_WeakPassword(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/account",
		`{"email":"dr@clinic.local","password":"short"}`, adminToken(t))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestDoctorCreateAccount_DoctorNotFound(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/999/account",
		`{"email":"dr@clinic.local","password":"ValidPass1!"}`, adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDoctorCreateAccount_MissingFields(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	cases := []struct{ body string }{
		{`{"email":"dr@clinic.local"}`},
		{`{"password":"ValidPass1!"}`},
		{`{}`},
	}
	for _, tc := range cases {
		rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/account", tc.body, adminToken(t))
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	}
}

func TestDoctorCreateAccount_DoctorForbidden(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/account",
		`{"email":"dr@clinic.local","password":"ValidPass1!"}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — SetDirections —

func TestDoctorSetDirections_Success(t *testing.T) {
	dw := sampleDW()
	dir := sampleDir()
	router := newDoctorRouter(
		&mockDoctorRepo{doctor: &dw},
		&mockDirectionRepo{direction: &dir},
	)

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/directions",
		`{"direction_ids":[1]}`, adminToken(t))

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDoctorSetDirections_Empty(t *testing.T) {
	dw := sampleDW()
	router := newDoctorRouter(&mockDoctorRepo{doctor: &dw}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/directions",
		`{"direction_ids":[]}`, adminToken(t))

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDoctorSetDirections_DoctorNotFound(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{err: apperrors.ErrNotFound}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/999/directions",
		`{"direction_ids":[1]}`, adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDoctorSetDirections_DoctorForbidden(t *testing.T) {
	router := newDoctorRouter(&mockDoctorRepo{}, &mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/directions",
		`{"direction_ids":[]}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
