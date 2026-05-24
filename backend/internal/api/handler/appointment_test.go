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

const testBotSecret = "test-bot-secret-key"

// mockApptRepo implements repository.AppointmentRepository for handler-layer tests.
type mockApptRepo struct {
	appt      *model.Appointment
	detail    *repository.AppointmentDetail
	list      []repository.AppointmentDetail
	err       error
	updateErr error
}

func (m *mockApptRepo) Create(_ context.Context, _ repository.CreateAppointmentInput) (*model.Appointment, error) {
	return m.appt, m.err
}

func (m *mockApptRepo) GetByID(_ context.Context, _ int64) (*repository.AppointmentDetail, error) {
	return m.detail, m.err
}

func (m *mockApptRepo) List(_ context.Context, _ repository.AppointmentFilter) ([]repository.AppointmentDetail, error) {
	return m.list, m.err
}

func (m *mockApptRepo) UpdateStatus(_ context.Context, _ int64, _, _ model.AppointmentStatus, _ *int64, _ *string) error {
	return m.updateErr
}

// helpers

func sampleApptResult() *model.Appointment {
	return &model.Appointment{
		ID: 1, DoctorID: 1, ServiceID: 1,
		Status:    model.StatusCreated,
		Source:    model.SourceAdminPanel,
		StartAt:   time.Now().Add(2 * time.Hour),
		EndAt:     time.Now().Add(3 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// sampleAdminDetail has full patient info, DoctorID=1.
func sampleAdminDetail() *repository.AppointmentDetail {
	tid := int64(12345)
	return &repository.AppointmentDetail{
		Appointment:       *sampleApptResult(),
		PatientName:       "Ivan Ivanov",
		PatientPhone:      "+79001234567",
		PatientTelegramID: &tid,
		DoctorFullName:    "John Smith",
		ServiceName:       "Consultation",
	}
}

// sampleDoctorDetail has DoctorID=0 to match mockDoctorRepo.GetDoctorIDByUserID return value.
func sampleDoctorDetail(status model.AppointmentStatus) *repository.AppointmentDetail {
	d := sampleAdminDetail()
	d.DoctorID = 0
	d.Status = status
	return d
}

// sampleAdminDetailWithStatus returns a detail in the given status for transition tests.
func sampleAdminDetailWithStatus(status model.AppointmentStatus) *repository.AppointmentDetail {
	d := sampleAdminDetail()
	d.Status = status
	return d
}

func activeSvcForAppt() *model.Service {
	return &model.Service{
		ID: 1, DoctorID: 1, DirectionID: 1,
		Name: "Consultation", DurationMinutes: 30,
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func activeDoctorForAppt() *model.DoctorWithDirections {
	return &model.DoctorWithDirections{
		Doctor: model.Doctor{
			ID: 1, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		Directions: []model.Direction{{ID: 1, Name: "Card", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}},
	}
}

func futureStartAt() string {
	return time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
}

func newAppointmentRouter(apptRepo *mockApptRepo, docRepo *mockDoctorRepo, svcRepo *mockServiceRepo, botSecret string) http.Handler {
	svc := service.NewAppointmentService(apptRepo, docRepo, svcRepo)
	h := NewAppointmentHandler(svc)

	r := chi.NewRouter()

	// Bot routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.BotAuth(botSecret))
		r.Post("/api/v1/bot/appointments", h.BotCreate)
		r.Post("/api/v1/bot/appointments/{id}/cancel", h.BotCancel)
	})

	// JWT routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("doctor"))
			r.Get("/api/v1/doctor/appointments", h.DoctorList)
			r.Get("/api/v1/doctor/appointments/{id}", h.DoctorGetByID)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/api/v1/appointments", h.AdminCreate)
			r.Get("/api/v1/appointments", h.List)
			r.Get("/api/v1/appointments/{id}", h.GetByID)
			r.Post("/api/v1/appointments/{id}/confirm", h.Confirm)
			r.Post("/api/v1/appointments/{id}/cancel", h.AdminCancel)
			r.Post("/api/v1/appointments/{id}/complete", h.Complete)
			r.Post("/api/v1/appointments/{id}/no-show", h.MarkNoShow)
		})
	})
	return r
}

func botReq(router http.Handler, method, path, body, botSecret string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if botSecret != "" {
		req.Header.Set("X-Bot-Token", botSecret)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// — Bot middleware —

func TestBotAuth_NoHeader(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments", "{}", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestBotAuth_WrongToken(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments", "{}", "wrong-secret")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestBotAuth_EmptySecret_RejectsAll(t *testing.T) {
	// misconfigured server: empty bot secret → all requests rejected
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, "")
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments", "{}", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — BotCreate —

func TestBotCreate_Success(t *testing.T) {
	doc := activeDoctorForAppt()
	router := newAppointmentRouter(
		&mockApptRepo{appt: sampleApptResult()},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvcForAppt()},
		testBotSecret,
	)

	body := `{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments", body, testBotSecret)

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Appointment
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.ID)
}

func TestBotCreate_MissingFields(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)

	cases := []string{
		`{"patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`,
		`{"patient_name":"Ivan","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`,
		`{"patient_name":"Ivan","patient_phone":"+7900","service_id":1,"start_at":"` + futureStartAt() + `"}`,
		`{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"start_at":"` + futureStartAt() + `"}`,
		`{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1}`,
	}
	for _, body := range cases {
		rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments", body, testBotSecret)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	}
}

func TestBotCreate_InvalidStartAt(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments",
		`{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"not-a-date"}`,
		testBotSecret)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBotCreate_DoctorInactive(t *testing.T) {
	inactive := activeDoctorForAppt()
	inactive.IsActive = false
	router := newAppointmentRouter(
		&mockApptRepo{},
		&mockDoctorRepo{doctor: inactive},
		&mockServiceRepo{svc: activeSvcForAppt()},
		testBotSecret,
	)
	body := `{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments", body, testBotSecret)
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestBotCreate_SlotTaken(t *testing.T) {
	doc := activeDoctorForAppt()
	router := newAppointmentRouter(
		&mockApptRepo{err: apperrors.ErrSlotTaken},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvcForAppt()},
		testBotSecret,
	)
	body := `{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments", body, testBotSecret)
	assert.Equal(t, http.StatusConflict, rr.Code)
}

// — BotCancel —

func TestBotCancel_Success(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleDoctorDetail(model.StatusCreated)},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments/1/cancel", "", testBotSecret)
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestBotCancel_NotFound(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{err: apperrors.ErrNotFound},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments/999/cancel", "", testBotSecret)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestBotCancel_InvalidTransition(t *testing.T) {
	// Patient cannot cancel a completed appointment
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleDoctorDetail(model.StatusCompleted)},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := botReq(router, http.MethodPost, "/api/v1/bot/appointments/1/cancel", "", testBotSecret)
	assert.Equal(t, http.StatusConflict, rr.Code)
}

// — AdminCreate —

func TestAdminCreate_Success(t *testing.T) {
	doc := activeDoctorForAppt()
	router := newAppointmentRouter(
		&mockApptRepo{appt: sampleApptResult()},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvcForAppt()},
		testBotSecret,
	)
	body := `{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments", body, adminToken(t))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp model.Appointment
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.ID)
}

func TestAdminCreate_DoctorForbidden(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	body := `{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments", body, doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminCreate_NoToken(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	body := `{"patient_name":"Ivan","patient_phone":"+7900","doctor_id":1,"service_id":1,"start_at":"` + futureStartAt() + `"}`
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments", body, "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — List (admin) —

func TestAdminList_Success(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{list: []repository.AppointmentDetail{*sampleAdminDetail()}},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []repository.AppointmentDetail
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestAdminList_WithFilters(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{list: []repository.AppointmentDetail{}},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments?doctor_id=1&status=created&limit=10&offset=0", "", adminToken(t))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminList_InvalidLimit(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	cases := []string{"0", "201", "-1", "abc"}
	for _, v := range cases {
		rr := bearerReq(router, http.MethodGet, "/api/v1/appointments?limit="+v, "", adminToken(t))
		assert.Equal(t, http.StatusBadRequest, rr.Code, "limit=%s", v)
	}
}

func TestAdminList_DoctorForbidden(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments", "", doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — GetByID (admin) —

func TestAdminGetByID_Success(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleAdminDetail()},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments/1", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	// Admin response includes patient_phone
	assert.Contains(t, resp, "patient_phone")
}

func TestAdminGetByID_NotFound(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{err: apperrors.ErrNotFound},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments/999", "", adminToken(t))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// — Status transitions (admin) — handler delegates to service, knows no transitions —

func TestAdminConfirm_Success(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleAdminDetailWithStatus(model.StatusCreated)},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments/1/confirm", "", adminToken(t))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAdminConfirm_InvalidTransition(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleAdminDetailWithStatus(model.StatusCompleted)},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments/1/confirm", "", adminToken(t))
	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestAdminCancel_Success(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleAdminDetailWithStatus(model.StatusConfirmed)},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments/1/cancel", `{"comment":"no reason"}`, adminToken(t))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAdminCancel_NotFound(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{err: apperrors.ErrNotFound},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments/999/cancel", "", adminToken(t))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAdminComplete_Success(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleAdminDetailWithStatus(model.StatusConfirmed)},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments/1/complete", "", adminToken(t))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAdminMarkNoShow_Success(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{detail: sampleAdminDetailWithStatus(model.StatusConfirmed)},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments/1/no-show", "", adminToken(t))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAdminComplete_DoctorForbidden(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodPost, "/api/v1/appointments/1/complete", "", doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — DoctorList —

func TestDoctorApptList_Success(t *testing.T) {
	detail := sampleDoctorDetail(model.StatusCreated)
	router := newAppointmentRouter(
		&mockApptRepo{list: []repository.AppointmentDetail{*detail}},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments", "", doctorToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp []map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestDoctorList_PrivacyTrimming(t *testing.T) {
	detail := sampleDoctorDetail(model.StatusCreated)
	// detail has patient_phone and patient_telegram_id set
	router := newAppointmentRouter(
		&mockApptRepo{list: []repository.AppointmentDetail{*detail}},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments", "", doctorToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	bodyStr := rr.Body.String()
	assert.NotContains(t, bodyStr, "patient_phone", "doctor response must not contain patient_phone")
	assert.NotContains(t, bodyStr, "patient_telegram_id", "doctor response must not contain patient_telegram_id")
	assert.Contains(t, bodyStr, "patient_name")
}

func TestDoctorList_QueryParamDoctorIDIgnored(t *testing.T) {
	detail := sampleDoctorDetail(model.StatusCreated)
	router := newAppointmentRouter(
		&mockApptRepo{list: []repository.AppointmentDetail{*detail}},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	// doctor_id=99 in query — must be silently ignored, doctor gets own appointments
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments?doctor_id=99", "", doctorToken(t))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDoctorList_AdminForbidden(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments", "", adminToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestDoctorApptList_NoToken(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments", "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestDoctorList_DoctorProfileNotFound(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{},
		&mockDoctorRepo{err: apperrors.ErrNotFound},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments", "", doctorToken(t))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// — DoctorGetByID —

func TestDoctorApptGetByID_Success(t *testing.T) {
	// DoctorID=0 in detail matches mockDoctorRepo.GetDoctorIDByUserID return value (0)
	detail := sampleDoctorDetail(model.StatusCreated)
	router := newAppointmentRouter(
		&mockApptRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments/1", "", doctorToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	bodyStr := rr.Body.String()
	assert.NotContains(t, bodyStr, "patient_phone")
	assert.NotContains(t, bodyStr, "patient_telegram_id")
}

func TestDoctorGetByID_Forbidden(t *testing.T) {
	// DoctorID=1 in detail != 0 returned by mock → forbidden
	detail := sampleAdminDetail() // DoctorID=1
	router := newAppointmentRouter(
		&mockApptRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments/1", "", doctorToken(t))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestDoctorApptGetByID_NotFound(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{err: apperrors.ErrNotFound},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/doctor/appointments/999", "", doctorToken(t))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// — Admin list includes patient data, doctor list does not —

func TestAdminList_IncludesPatientPhone(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{list: []repository.AppointmentDetail{*sampleAdminDetail()}},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments", "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)
	bodyStr := rr.Body.String()
	assert.Contains(t, bodyStr, "patient_phone", "admin response must include patient_phone")
	assert.Contains(t, bodyStr, "patient_telegram_id", "admin response must include patient_telegram_id")
}

// — parseAppointmentFilter edge cases —

func TestAdminList_InvalidDoctorID(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments?doctor_id=abc", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminList_InvalidDateFrom(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments?date_from=not-a-date", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminList_InvalidOffset(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments?offset=-1", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminList_BranchIDFilter_OK(t *testing.T) {
	router := newAppointmentRouter(
		&mockApptRepo{list: []repository.AppointmentDetail{}},
		&mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret,
	)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments?branch_id=1", "", adminToken(t))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminList_BranchIDFilter_Invalid(t *testing.T) {
	router := newAppointmentRouter(&mockApptRepo{}, &mockDoctorRepo{}, &mockServiceRepo{}, testBotSecret)
	rr := bearerReq(router, http.MethodGet, "/api/v1/appointments?branch_id=abc", "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
