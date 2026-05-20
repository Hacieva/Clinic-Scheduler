package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/availability"
)

type AvailabilityHandler struct {
	svc *availability.Service
}

func NewAvailabilityHandler(svc *availability.Service) *AvailabilityHandler {
	return &AvailabilityHandler{svc: svc}
}

type availabilityResponse struct {
	DoctorID               int64             `json:"doctor_id"`
	ServiceID              int64             `json:"service_id"`
	ServiceDurationMinutes int               `json:"service_duration_minutes"`
	Availability           []dayAvailability `json:"availability"`
}

type dayAvailability struct {
	Date  string   `json:"date"`  // "YYYY-MM-DD"
	Slots []string `json:"slots"` // ["HH:MM", ...]
}

func (h *AvailabilityHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseQueryInt64(w, r, "doctor_id")
	if !ok {
		return
	}
	serviceID, ok := parseQueryInt64(w, r, "service_id")
	if !ok {
		return
	}
	from, ok := parseQueryDate(w, r, "date_from")
	if !ok {
		return
	}
	to, ok := parseQueryDate(w, r, "date_to")
	if !ok {
		return
	}

	durationMin, err := h.svc.GetServiceDuration(r.Context(), serviceID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "service not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	days, err := h.svc.GetAvailability(r.Context(), doctorID, serviceID, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := availabilityResponse{
		DoctorID:               doctorID,
		ServiceID:              serviceID,
		ServiceDurationMinutes: durationMin,
		Availability:           make([]dayAvailability, 0, len(days)),
	}
	for _, d := range days {
		da := dayAvailability{
			Date:  d.Date.Format("2006-01-02"),
			Slots: make([]string, 0, len(d.Slots)),
		}
		for _, s := range d.Slots {
			da.Slots = append(da.Slots, s.Start.Format("15:04"))
		}
		resp.Availability = append(resp.Availability, da)
	}

	writeJSON(w, http.StatusOK, resp)
}

func parseQueryInt64(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": name + " is required"})
		return 0, false
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid " + name})
		return 0, false
	}
	return v, true
}

func parseQueryDate(w http.ResponseWriter, r *http.Request, name string) (time.Time, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": name + " is required"})
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid " + name + ", expected YYYY-MM-DD"})
		return time.Time{}, false
	}
	return t, true
}
