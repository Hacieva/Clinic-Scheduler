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
	"github.com/Hacieva/clinic-scheduler/backend/internal/availability"
)

// — mocks for availability.*Repository interfaces —

type mockAvailScheduleRepo struct {
	schedule []availability.RegularSchedule
	excepts  []availability.Exception
	err      error
}

func (m *mockAvailScheduleRepo) GetWorkingHours(_ context.Context, _ int64) ([]availability.RegularSchedule, error) {
	return m.schedule, m.err
}

func (m *mockAvailScheduleRepo) GetScheduleExceptions(_ context.Context, _ int64, _, _ time.Time) ([]availability.Exception, error) {
	return m.excepts, m.err
}

type mockAvailApptRepo struct {
	slots []availability.Slot
	err   error
}

func (m *mockAvailApptRepo) GetSlotsByDoctor(_ context.Context, _ int64, _, _ time.Time) ([]availability.Slot, error) {
	return m.slots, m.err
}

type mockAvailServiceRepo struct {
	duration int
	err      error
}

func (m *mockAvailServiceRepo) GetDurationMinutes(_ context.Context, _ int64) (int, error) {
	return m.duration, m.err
}

// — router builder —

func newAvailabilityRouter(
	schedRepo *mockAvailScheduleRepo,
	apptRepo *mockAvailApptRepo,
	svcRepo *mockAvailServiceRepo,
) http.Handler {
	svc := availability.NewService(schedRepo, apptRepo, svcRepo)
	h := NewAvailabilityHandler(svc)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Get("/api/v1/availability", h.GetAvailability)
	})
	return r
}

// wedScheduleHandler is a Wednesday 09:00–12:00 working schedule for handler tests.
var wedScheduleHandler = []availability.RegularSchedule{
	{
		DayOfWeek: time.Wednesday,
		Start:     time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
		End:       time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC),
	},
}

// refWed is a Wednesday date used across handler tests.
var refWed = "2026-05-20" // Wednesday

// — success cases —

func TestGetAvailability_Success(t *testing.T) {
	router := newAvailabilityRouter(
		&mockAvailScheduleRepo{schedule: wedScheduleHandler},
		&mockAvailApptRepo{},
		&mockAvailServiceRepo{duration: 60},
	)

	url := "/api/v1/availability?doctor_id=1&service_id=2&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))

	require.Equal(t, http.StatusOK, rr.Code)

	var resp availabilityResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	assert.Equal(t, int64(1), resp.DoctorID)
	assert.Equal(t, int64(2), resp.ServiceID)
	assert.Equal(t, 60, resp.ServiceDurationMinutes)
	require.Len(t, resp.Availability, 1)
	assert.Equal(t, refWed, resp.Availability[0].Date)
	// 09:00–12:00 with 60min service, 30min step → slots: 09:00, 09:30, 10:00, 10:30, 11:00
	assert.Len(t, resp.Availability[0].Slots, 5)
	assert.Equal(t, "09:00", resp.Availability[0].Slots[0])
	assert.Equal(t, "11:00", resp.Availability[0].Slots[4])
}

func TestGetAvailability_SlotsInHHMMFormat(t *testing.T) {
	router := newAvailabilityRouter(
		&mockAvailScheduleRepo{schedule: wedScheduleHandler},
		&mockAvailApptRepo{},
		&mockAvailServiceRepo{duration: 30},
	)

	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp availabilityResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Availability)
	for _, slot := range resp.Availability[0].Slots {
		// Each slot must match HH:MM
		_, err := time.Parse("15:04", slot)
		assert.NoError(t, err, "slot %q is not HH:MM", slot)
	}
}

func TestGetAvailability_NoWorkingDays_EmptyAvailability(t *testing.T) {
	// Schedule only has Wednesday, range is Thursday only.
	thu := "2026-05-21"
	router := newAvailabilityRouter(
		&mockAvailScheduleRepo{schedule: wedScheduleHandler},
		&mockAvailApptRepo{},
		&mockAvailServiceRepo{duration: 60},
	)

	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=" + thu + "&date_to=" + thu
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp availabilityResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Empty(t, resp.Availability)
	assert.Equal(t, 60, resp.ServiceDurationMinutes)
}

func TestGetAvailability_BookedSlotExcluded(t *testing.T) {
	// Book 09:00–10:00 → that slot must not appear.
	booked := []availability.Slot{
		{
			Start: time.Date(2026, 5, 20, 9, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		},
	}
	router := newAvailabilityRouter(
		&mockAvailScheduleRepo{schedule: wedScheduleHandler},
		&mockAvailApptRepo{slots: booked},
		&mockAvailServiceRepo{duration: 60},
	)

	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp availabilityResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp.Availability, 1)
	for _, slot := range resp.Availability[0].Slots {
		assert.NotEqual(t, "09:00", slot, "booked slot 09:00 must not appear")
	}
}

func TestGetAvailability_DayOff(t *testing.T) {
	refDate := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	excepts := []availability.Exception{{Date: refDate, Type: "day_off"}}
	router := newAvailabilityRouter(
		&mockAvailScheduleRepo{schedule: wedScheduleHandler, excepts: excepts},
		&mockAvailApptRepo{},
		&mockAvailServiceRepo{duration: 60},
	)

	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp availabilityResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Empty(t, resp.Availability)
}

func TestGetAvailability_DoctorTokenAllowed(t *testing.T) {
	router := newAvailabilityRouter(
		&mockAvailScheduleRepo{schedule: wedScheduleHandler},
		&mockAvailApptRepo{},
		&mockAvailServiceRepo{duration: 30},
	)
	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", doctorToken(t))
	assert.Equal(t, http.StatusOK, rr.Code)
}

// — auth cases —

func TestGetAvailability_NoToken(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — missing / invalid query param cases —

func TestGetAvailability_MissingDoctorID(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "doctor_id")
}

func TestGetAvailability_MissingServiceID(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?doctor_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetAvailability_MissingDateFrom(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?doctor_id=1&service_id=1&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetAvailability_MissingDateTo(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetAvailability_InvalidDoctorID(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?doctor_id=abc&service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetAvailability_ZeroDoctorID(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?doctor_id=0&service_id=1&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetAvailability_InvalidDateFormat(t *testing.T) {
	router := newAvailabilityRouter(&mockAvailScheduleRepo{}, &mockAvailApptRepo{}, &mockAvailServiceRepo{duration: 30})
	url := "/api/v1/availability?doctor_id=1&service_id=1&date_from=20-05-2026&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// — response structure —

func TestGetAvailability_ResponseStructure(t *testing.T) {
	router := newAvailabilityRouter(
		&mockAvailScheduleRepo{schedule: wedScheduleHandler},
		&mockAvailApptRepo{},
		&mockAvailServiceRepo{duration: 30},
	)

	url := "/api/v1/availability?doctor_id=7&service_id=3&date_from=" + refWed + "&date_to=" + refWed
	rr := bearerReq(router, http.MethodGet, url, "", adminToken(t))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp availabilityResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	assert.Equal(t, int64(7), resp.DoctorID)
	assert.Equal(t, int64(3), resp.ServiceID)
	assert.Equal(t, 30, resp.ServiceDurationMinutes)
	assert.NotNil(t, resp.Availability)
}
