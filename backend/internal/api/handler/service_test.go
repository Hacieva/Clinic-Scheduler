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

// mockServiceRepo implements repository.ServiceRepository for handler-layer tests.
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
		ID: 1, DoctorID: input.DoctorID, DirectionID: input.DirectionID,
		Name: input.Name, Description: input.Description,
		DurationMinutes: input.DurationMinutes, Price: input.Price,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockServiceRepo) Update(_ context.Context, id int64, input repository.UpdateServiceInput) (*model.Service, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Service{
		ID: id, DirectionID: input.DirectionID, Name: input.Name,
		Description: input.Description, DurationMinutes: input.DurationMinutes, Price: input.Price,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
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

func sampleServiceRow() *model.Service {
	price := int64(150000)
	return &model.Service{
		ID: 1, DoctorID: 1, DirectionID: 1, Name: "Consultation",
		DurationMinutes: 30, Price: &price,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// sampleDoctorWithOneDirection returns a doctor with direction ID=1.
func sampleDoctorWithOneDirection() *model.DoctorWithDirections {
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

func newServiceRouter(svcRepo *mockServiceRepo, docRepo *mockDoctorRepo) http.Handler {
	svc := service.NewMedicalServiceService(svcRepo, docRepo)
	h := NewServiceHandler(svc)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Get("/api/v1/doctors/{id}/services", h.List)
		r.Get("/api/v1/doctors/{id}/services/{serviceId}", h.GetByID)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/api/v1/doctors/{id}/services", h.Create)
			r.Put("/api/v1/doctors/{id}/services/{serviceId}", h.Update)
			r.Delete("/api/v1/doctors/{id}/services/{serviceId}", h.Delete)
		})
	})
	return r
}

// — List —

func TestServiceList_Success(t *testing.T) {
	router := newServiceRouter(
		&mockServiceRepo{services: []model.Service{*sampleServiceRow()}},
		&mockDoctorRepo{},
	)

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/services", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.Service
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "Consultation", resp[0].Name)
}

func TestServiceList_Empty(t *testing.T) {
	router := newServiceRouter(
		&mockServiceRepo{services: []model.Service{}},
		&mockDoctorRepo{},
	)

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/services", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.Service
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Empty(t, resp)
}

func TestServiceList_DoctorAllowed(t *testing.T) {
	router := newServiceRouter(
		&mockServiceRepo{services: []model.Service{}},
		&mockDoctorRepo{},
	)

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/services", "", doctorToken(t))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestServiceList_NoToken(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/services", "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — GetByID —

func TestServiceGetByID_Success(t *testing.T) {
	s := sampleServiceRow()
	router := newServiceRouter(&mockServiceRepo{svc: s}, &mockDoctorRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/services/1", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.Service
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "Consultation", resp.Name)
}

func TestServiceGetByID_NotFound(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{err: apperrors.ErrNotFound}, &mockDoctorRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/services/999", "", adminToken(t))

	require.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "service not found", resp["error"])
}

func TestServiceGetByID_InvalidServiceID(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/services/abc", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// — Create —

func TestServiceCreate_Success(t *testing.T) {
	dw := sampleDoctorWithOneDirection()
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/services",
		`{"direction_id":1,"name":"Consultation","duration_minutes":30}`, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Service
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Consultation", resp.Name)
}

func TestServiceCreate_WithPrice(t *testing.T) {
	dw := sampleDoctorWithOneDirection()
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/services",
		`{"direction_id":1,"name":"Premium","duration_minutes":60,"price":300000}`, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Service
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotNil(t, resp.Price)
	assert.Equal(t, int64(300000), *resp.Price)
}

func TestServiceCreate_MissingFields(t *testing.T) {
	dw := sampleDoctorWithOneDirection()
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	cases := []struct{ body string }{
		{`{"direction_id":1,"duration_minutes":30}`},  // missing name
		{`{"name":"X","duration_minutes":30}`},        // missing direction_id
		{`{"direction_id":1,"name":"X"}`},             // missing duration_minutes
		{`{}`},
	}
	for _, tc := range cases {
		rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/services", tc.body, adminToken(t))
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	}
}

func TestServiceCreate_DirectionMismatch(t *testing.T) {
	// Doctor has direction 2, input requests direction 1.
	dw := &model.DoctorWithDirections{
		Doctor: model.Doctor{ID: 1, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Directions: []model.Direction{
			{ID: 2, Name: "Neurology", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
	}
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/services",
		`{"direction_id":1,"name":"Consultation","duration_minutes":30}`, adminToken(t))

	require.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "direction does not belong to doctor", resp["error"])
}

func TestServiceCreate_DoctorNotFound(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{err: apperrors.ErrNotFound})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/999/services",
		`{"direction_id":1,"name":"X","duration_minutes":30}`, adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServiceCreate_DoctorForbidden(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/services",
		`{"direction_id":1,"name":"X","duration_minutes":30}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestServiceCreate_NoToken(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/services",
		`{"direction_id":1,"name":"X","duration_minutes":30}`, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — Update —

func TestServiceUpdate_Success(t *testing.T) {
	dw := sampleDoctorWithOneDirection()
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/services/1",
		`{"direction_id":1,"name":"Updated","duration_minutes":45}`, adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.Service
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Updated", resp.Name)
}

func TestServiceUpdate_NotFound(t *testing.T) {
	dw := sampleDoctorWithOneDirection()
	router := newServiceRouter(&mockServiceRepo{err: apperrors.ErrNotFound}, &mockDoctorRepo{doctor: dw})

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/services/999",
		`{"direction_id":1,"name":"X","duration_minutes":30}`, adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServiceUpdate_DirectionMismatch(t *testing.T) {
	dw := &model.DoctorWithDirections{
		Doctor: model.Doctor{ID: 1, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Directions: []model.Direction{
			{ID: 2, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
	}
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{doctor: dw})

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/services/1",
		`{"direction_id":1,"name":"X","duration_minutes":30}`, adminToken(t))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestServiceUpdate_DoctorForbidden(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})

	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/services/1",
		`{"direction_id":1,"name":"X","duration_minutes":30}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — Delete —

func TestServiceDelete_Success(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/services/1", "", adminToken(t))

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestServiceDelete_NotFound(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{err: apperrors.ErrNotFound}, &mockDoctorRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/services/999", "", adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServiceDelete_DoctorForbidden(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/services/1", "", doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestServiceDelete_NoToken(t *testing.T) {
	router := newServiceRouter(&mockServiceRepo{}, &mockDoctorRepo{})
	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/services/1", "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
