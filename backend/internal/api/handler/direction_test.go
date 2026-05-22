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
	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

// mockDirectionRepo satisfies repository.DirectionRepository.
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
	return &model.Direction{
		ID: 1, Name: name, Description: description,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockDirectionRepo) Update(_ context.Context, id int64, name string, description *string) (*model.Direction, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.Direction{
		ID: id, Name: name, Description: description,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (m *mockDirectionRepo) SoftDelete(_ context.Context, _ int64) error {
	return m.err
}

func newDirectionRouter(repo *mockDirectionRepo) http.Handler {
	svc := service.NewDirectionService(repo)
	h := NewDirectionHandler(svc)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Get("/api/v1/directions", h.List)
		r.Get("/api/v1/directions/{id}", h.GetByID)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/api/v1/directions", h.Create)
			r.Put("/api/v1/directions/{id}", h.Update)
			r.Delete("/api/v1/directions/{id}", h.Delete)
		})
	})
	return r
}

func sampleDir() model.Direction {
	desc := "test desc"
	return model.Direction{
		ID: 1, Name: "Cardiology", Description: &desc,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func adminToken(t *testing.T) string {
	t.Helper()
	tok, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)
	return tok
}

func doctorToken(t *testing.T) string {
	t.Helper()
	tok, err := auth.GenerateAccessToken(43, model.RoleDoctor, nil, testSecret)
	require.NoError(t, err)
	return tok
}

// — List —

func TestDirectionList_Success(t *testing.T) {
	dirs := []model.Direction{sampleDir()}
	router := newDirectionRouter(&mockDirectionRepo{directions: dirs})

	rr := bearerReq(router, http.MethodGet, "/api/v1/directions", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.Direction
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "Cardiology", resp[0].Name)
}

func TestDirectionList_Empty(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{directions: []model.Direction{}})

	rr := bearerReq(router, http.MethodGet, "/api/v1/directions", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.Direction
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Empty(t, resp)
}

func TestDirectionList_NoToken(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})
	rr := bearerReq(router, http.MethodGet, "/api/v1/directions", "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestDirectionList_DoctorAllowed(t *testing.T) {
	dirs := []model.Direction{sampleDir()}
	router := newDirectionRouter(&mockDirectionRepo{directions: dirs})

	rr := bearerReq(router, http.MethodGet, "/api/v1/directions", "", doctorToken(t))

	assert.Equal(t, http.StatusOK, rr.Code)
}

// — GetByID —

func TestDirectionGetByID_Success(t *testing.T) {
	d := sampleDir()
	router := newDirectionRouter(&mockDirectionRepo{direction: &d})

	rr := bearerReq(router, http.MethodGet, "/api/v1/directions/1", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.Direction
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.ID)
}

func TestDirectionGetByID_NotFound(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{err: apperrors.ErrNotFound})

	rr := bearerReq(router, http.MethodGet, "/api/v1/directions/999", "", adminToken(t))

	require.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "direction not found", resp["error"])
}

func TestDirectionGetByID_InvalidID(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/directions/abc", "", adminToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// — Create —

func TestDirectionCreate_Success(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/directions",
		`{"name":"Neurology","description":"brain stuff"}`, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Direction
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Neurology", resp.Name)
}

func TestDirectionCreate_MissingName(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/directions",
		`{"description":"no name"}`, adminToken(t))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "name is required", resp["error"])
}

func TestDirectionCreate_MalformedJSON(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/directions", `{not json`, adminToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDirectionCreate_DoctorForbidden(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/directions",
		`{"name":"Neurology"}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestDirectionCreate_NoToken(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})
	rr := bearerReq(router, http.MethodPost, "/api/v1/directions", `{"name":"X"}`, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — Update —

func TestDirectionUpdate_Success(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPut, "/api/v1/directions/1",
		`{"name":"Updated"}`, adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.Direction
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Updated", resp.Name)
}

func TestDirectionUpdate_NotFound(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{err: apperrors.ErrNotFound})

	rr := bearerReq(router, http.MethodPut, "/api/v1/directions/999",
		`{"name":"X"}`, adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDirectionUpdate_MissingName(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPut, "/api/v1/directions/1", `{}`, adminToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDirectionUpdate_DoctorForbidden(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodPut, "/api/v1/directions/1",
		`{"name":"X"}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — Delete —

func TestDirectionDelete_Success(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/directions/1", "", adminToken(t))

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDirectionDelete_NotFound(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{err: apperrors.ErrNotFound})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/directions/999", "", adminToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDirectionDelete_DoctorForbidden(t *testing.T) {
	router := newDirectionRouter(&mockDirectionRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/directions/1", "", doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
