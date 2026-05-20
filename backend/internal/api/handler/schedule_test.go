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

// mockScheduleRepo implements repository.ScheduleRepository for handler-layer tests.
type mockScheduleRepo struct {
	workingHours []model.WorkingHours
	exceptions   []model.ScheduleException
	exception    *model.ScheduleException
	err          error
	replaceErr   error
}

func (m *mockScheduleRepo) ListWorkingHours(_ context.Context, _ int64) ([]model.WorkingHours, error) {
	return m.workingHours, m.err
}

func (m *mockScheduleRepo) ReplaceWorkingHours(_ context.Context, _ int64, _ []repository.CreateWorkingHoursInput) error {
	return m.replaceErr
}

func (m *mockScheduleRepo) ListExceptions(_ context.Context, _ int64, _, _ time.Time) ([]model.ScheduleException, error) {
	return m.exceptions, m.err
}

func (m *mockScheduleRepo) CreateException(_ context.Context, _ repository.CreateExceptionInput) (*model.ScheduleException, error) {
	return m.exception, m.err
}

func (m *mockScheduleRepo) UpdateException(_ context.Context, _ int64, _ repository.CreateExceptionInput) (*model.ScheduleException, error) {
	return m.exception, m.err
}

func (m *mockScheduleRepo) DeleteException(_ context.Context, _ int64) error {
	return m.err
}

func newScheduleRouter(repo *mockScheduleRepo) http.Handler {
	svc := service.NewScheduleService(repo)
	h := NewScheduleHandler(svc)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Get("/api/v1/doctors/{id}/working-hours", h.ListWorkingHours)
		r.Get("/api/v1/doctors/{id}/exceptions", h.ListExceptions)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Put("/api/v1/doctors/{id}/working-hours", h.ReplaceWorkingHours)
			r.Post("/api/v1/doctors/{id}/exceptions", h.CreateException)
			r.Put("/api/v1/doctors/{id}/exceptions/{exId}", h.UpdateException)
			r.Delete("/api/v1/doctors/{id}/exceptions/{exId}", h.DeleteException)
		})
	})
	return r
}

func sampleWorkingHours() model.WorkingHours {
	return model.WorkingHours{
		ID: 1, DoctorID: 1, DayOfWeek: 1,
		StartTime: time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2000, 1, 1, 17, 0, 0, 0, time.UTC),
		IsActive:  true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func sampleException(exType model.ExceptionType) *model.ScheduleException {
	ex := &model.ScheduleException{
		ID: 1, DoctorID: 1,
		Date:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Type:      exType,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if exType == model.ExceptionTypeCustomWorkingHours {
		start := time.Date(2000, 1, 1, 10, 0, 0, 0, time.UTC)
		end := time.Date(2000, 1, 1, 14, 0, 0, 0, time.UTC)
		ex.StartTime = &start
		ex.EndTime = &end
	}
	return ex
}

// — ListWorkingHours —

func TestScheduleListWorkingHours_Success(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{
		workingHours: []model.WorkingHours{sampleWorkingHours()},
	})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/working-hours", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.WorkingHours
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, 1, resp[0].DayOfWeek)
}

func TestScheduleListWorkingHours_DoctorAllowed(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{workingHours: []model.WorkingHours{}})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/working-hours", "", doctorToken(t))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestScheduleListWorkingHours_NoToken(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/working-hours", "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — ReplaceWorkingHours —

func TestReplaceWorkingHours_Success(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"items":[{"day_of_week":1,"start_time":"09:00","end_time":"17:00"}]}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/working-hours", body, adminToken(t))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestReplaceWorkingHours_EmptyItems(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"items":[]}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/working-hours", body, adminToken(t))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestReplaceWorkingHours_InvalidDay(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"items":[{"day_of_week":0,"start_time":"09:00","end_time":"17:00"}]}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/working-hours", body, adminToken(t))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestReplaceWorkingHours_StartAfterEnd(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"items":[{"day_of_week":1,"start_time":"17:00","end_time":"09:00"}]}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/working-hours", body, adminToken(t))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestReplaceWorkingHours_InvalidTimeFormat(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"items":[{"day_of_week":1,"start_time":"9am","end_time":"17:00"}]}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/working-hours", body, adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReplaceWorkingHours_DoctorForbidden(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"items":[{"day_of_week":1,"start_time":"09:00","end_time":"17:00"}]}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/working-hours", body, doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestReplaceWorkingHours_NoToken(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"items":[{"day_of_week":1,"start_time":"09:00","end_time":"17:00"}]}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/working-hours", body, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — ListExceptions —

func TestListExceptions_Success(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{
		exceptions: []model.ScheduleException{*sampleException(model.ExceptionTypeDayOff)},
	})
	rr := bearerReq(router, http.MethodGet,
		"/api/v1/doctors/1/exceptions?from=2026-06-01&to=2026-06-30", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []model.ScheduleException
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestListExceptions_MissingParams(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctors/1/exceptions", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListExceptions_InvalidFromDate(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	rr := bearerReq(router, http.MethodGet,
		"/api/v1/doctors/1/exceptions?from=bad-date&to=2026-06-30", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListExceptions_DoctorAllowed(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{exceptions: []model.ScheduleException{}})
	rr := bearerReq(router, http.MethodGet,
		"/api/v1/doctors/1/exceptions?from=2026-06-01&to=2026-06-30", "", doctorToken(t))
	assert.Equal(t, http.StatusOK, rr.Code)
}

// — CreateException —

func TestCreateException_DayOff_Success(t *testing.T) {
	ex := sampleException(model.ExceptionTypeDayOff)
	router := newScheduleRouter(&mockScheduleRepo{exception: ex})
	body := `{"date":"2026-06-01","type":"day_off"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.ScheduleException
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, model.ExceptionTypeDayOff, resp.Type)
}

func TestCreateException_CustomWorkingHours_Success(t *testing.T) {
	ex := sampleException(model.ExceptionTypeCustomWorkingHours)
	router := newScheduleRouter(&mockScheduleRepo{exception: ex})
	body := `{"date":"2026-06-01","type":"custom_working_hours","start_time":"10:00","end_time":"14:00"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
}

func TestCreateException_DayOff_WithStartTime(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"date":"2026-06-01","type":"day_off","start_time":"09:00"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestCreateException_CustomWorkingHours_MissingStart(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"date":"2026-06-01","type":"custom_working_hours","end_time":"14:00"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestCreateException_InvalidType(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"date":"2026-06-01","type":"vacation"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestCreateException_MissingDate(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"type":"day_off"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateException_InvalidDateFormat(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"date":"01.06.2026","type":"day_off"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateException_Conflict(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{err: apperrors.ErrConflict})
	body := `{"date":"2026-06-01","type":"day_off"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, adminToken(t))
	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestCreateException_DoctorForbidden(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"date":"2026-06-01","type":"day_off"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/doctors/1/exceptions", body, doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — UpdateException —

func TestUpdateException_Success(t *testing.T) {
	ex := sampleException(model.ExceptionTypeDayOff)
	router := newScheduleRouter(&mockScheduleRepo{exception: ex})
	body := `{"date":"2026-06-02","type":"day_off"}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/exceptions/1", body, adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.ScheduleException
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, model.ExceptionTypeDayOff, resp.Type)
}

func TestUpdateException_NotFound(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{err: apperrors.ErrNotFound})
	body := `{"date":"2026-06-02","type":"day_off"}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/exceptions/999", body, adminToken(t))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUpdateException_InvalidExID(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"date":"2026-06-02","type":"day_off"}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/exceptions/abc", body, adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateException_DoctorForbidden(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	body := `{"date":"2026-06-02","type":"day_off"}`
	rr := bearerReq(router, http.MethodPut, "/api/v1/doctors/1/exceptions/1", body, doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — DeleteException —

func TestDeleteException_Success(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/exceptions/1", "", adminToken(t))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDeleteException_NotFound(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{err: apperrors.ErrNotFound})
	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/exceptions/999", "", adminToken(t))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteException_DoctorForbidden(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/exceptions/1", "", doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestDeleteException_NoToken(t *testing.T) {
	router := newScheduleRouter(&mockScheduleRepo{})
	rr := bearerReq(router, http.MethodDelete, "/api/v1/doctors/1/exceptions/1", "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
