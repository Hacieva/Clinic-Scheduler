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
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

// mockBranchRepo implements repository.BranchRepository without a real DB.
type mockBranchRepo struct {
	branches      []model.Branch
	branch        *model.Branch
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

func ownerToken(t *testing.T) string {
	t.Helper()
	tok, err := auth.GenerateAccessToken(44, model.RoleOwner, nil, testSecret)
	require.NoError(t, err)
	return tok
}

func newBranchRouter(repo *mockBranchRepo) http.Handler {
	svc := service.NewBranchService(repo)
	h := NewBranchHandler(svc)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("owner", "admin"))
			r.Get("/api/v1/branches", h.List)
			r.Get("/api/v1/branches/{id}", h.GetByID)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("owner"))
			r.Post("/api/v1/branches", h.Create)
			r.Patch("/api/v1/branches/{id}", h.Update)
			r.Delete("/api/v1/branches/{id}", h.Delete)
		})
	})
	return r
}

func sampleBranchModel() model.Branch {
	return model.Branch{
		ID: 1, Name: "Главный филиал", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// — List —

func TestBranchHandlerList_OwnerOK(t *testing.T) {
	b := sampleBranchModel()
	router := newBranchRouter(&mockBranchRepo{branches: []model.Branch{b}})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches", "", ownerToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.Branch
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "Главный филиал", resp[0].Name)
}

func TestBranchHandlerList_AdminOK(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{branches: []model.Branch{}})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches", "", adminToken(t))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestBranchHandlerList_DoctorForbidden(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches", "", doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestBranchHandlerList_NoToken(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches", "", "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — GetByID —

func TestBranchHandlerGetByID_OwnerOK(t *testing.T) {
	b := sampleBranchModel()
	router := newBranchRouter(&mockBranchRepo{branch: &b})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches/1", "", ownerToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.Branch
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.ID)
}

func TestBranchHandlerGetByID_AdminOK(t *testing.T) {
	b := sampleBranchModel()
	router := newBranchRouter(&mockBranchRepo{branch: &b})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches/1", "", adminToken(t))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestBranchHandlerGetByID_NotFound(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{err: apperrors.ErrNotFound})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches/999", "", ownerToken(t))

	require.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "branch not found", resp["error"])
}

func TestBranchHandlerGetByID_InvalidID(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodGet, "/api/v1/branches/abc", "", ownerToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// — Create —

func TestBranchHandlerCreate_OwnerOK(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/branches",
		`{"name":"Филиал №2"}`, ownerToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Branch
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Филиал №2", resp.Name)
}

func TestBranchHandlerCreate_EmptyName(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/branches",
		`{"name":""}`, ownerToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBranchHandlerCreate_WhitespaceName(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/branches",
		`{"name":"   "}`, ownerToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBranchHandlerCreate_MalformedJSON(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/branches", `{bad`, ownerToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBranchHandlerCreate_AdminForbidden(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/branches",
		`{"name":"Филиал №2"}`, adminToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestBranchHandlerCreate_DoctorForbidden(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/branches",
		`{"name":"Филиал №2"}`, doctorToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestBranchHandlerCreate_NoToken(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPost, "/api/v1/branches", `{"name":"X"}`, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — Update —

func TestBranchHandlerUpdate_OwnerOK(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPatch, "/api/v1/branches/1",
		`{"name":"Новое название"}`, ownerToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.Branch
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Новое название", resp.Name)
}

func TestBranchHandlerUpdate_EmptyName(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPatch, "/api/v1/branches/1",
		`{"name":""}`, ownerToken(t))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBranchHandlerUpdate_NotFound(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{err: apperrors.ErrNotFound})

	rr := bearerReq(router, http.MethodPatch, "/api/v1/branches/999",
		`{"name":"X"}`, ownerToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestBranchHandlerUpdate_AdminForbidden(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodPatch, "/api/v1/branches/1",
		`{"name":"X"}`, adminToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — Delete —

func TestBranchHandlerDelete_OwnerOK(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{hasActiveDocs: false})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/branches/1", "", ownerToken(t))

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestBranchHandlerDelete_HasActiveDoctors(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{hasActiveDocs: true})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/branches/1", "", ownerToken(t))

	require.Equal(t, http.StatusConflict, rr.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "branch has active doctors", resp["error"])
}

func TestBranchHandlerDelete_NotFound(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{hasActiveDocs: false, err: apperrors.ErrNotFound})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/branches/999", "", ownerToken(t))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestBranchHandlerDelete_AdminForbidden(t *testing.T) {
	router := newBranchRouter(&mockBranchRepo{})

	rr := bearerReq(router, http.MethodDelete, "/api/v1/branches/1", "", adminToken(t))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
