package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type ScheduleHandler struct {
	svc *service.ScheduleService
}

func NewScheduleHandler(svc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

// — Request types —

type workingHoursItemRequest struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"` // "HH:MM"
	EndTime   string `json:"end_time"`   // "HH:MM"
}

type replaceWorkingHoursRequest struct {
	Items []workingHoursItemRequest `json:"items"`
}

type exceptionRequest struct {
	Date      string  `json:"date"`       // "YYYY-MM-DD"
	Type      string  `json:"type"`       // "day_off" | "custom_working_hours"
	StartTime *string `json:"start_time"` // "HH:MM", nil for day_off
	EndTime   *string `json:"end_time"`   // "HH:MM", nil for day_off
	Comment   *string `json:"comment"`
}

// — Working hours handlers —

func (h *ScheduleHandler) ListWorkingHours(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListWorkingHours(r.Context(), doctorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ScheduleHandler) ReplaceWorkingHours(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req replaceWorkingHoursRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	inputs := make([]service.WorkingHoursInput, 0, len(req.Items))
	for _, item := range req.Items {
		start, err := time.Parse("15:04", item.StartTime)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid start_time format, expected HH:MM"})
			return
		}
		end, err := time.Parse("15:04", item.EndTime)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid end_time format, expected HH:MM"})
			return
		}
		inputs = append(inputs, service.WorkingHoursInput{
			DayOfWeek: item.DayOfWeek,
			StartTime: start,
			EndTime:   end,
		})
	}

	if err := h.svc.ReplaceWorkingHours(r.Context(), doctorID, inputs); err != nil {
		if errors.Is(err, apperrors.ErrInvalidSchedule) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid schedule parameters"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// — Exception handlers —

func (h *ScheduleHandler) ListExceptions(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from and to query params are required"})
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from date, expected YYYY-MM-DD"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid to date, expected YYYY-MM-DD"})
		return
	}

	items, err := h.svc.ListExceptions(r.Context(), doctorID, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ScheduleHandler) CreateException(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req exceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	inp, ok := parseExceptionInput(w, doctorID, req)
	if !ok {
		return
	}

	ex, err := h.svc.CreateException(r.Context(), inp)
	if err != nil {
		writeExceptionError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ex)
}

func (h *ScheduleHandler) UpdateException(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	exID, ok := parseExIDParam(w, r)
	if !ok {
		return
	}
	var req exceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	inp, ok := parseExceptionInput(w, doctorID, req)
	if !ok {
		return
	}

	ex, err := h.svc.UpdateException(r.Context(), exID, inp)
	if err != nil {
		writeExceptionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ex)
}

func (h *ScheduleHandler) DeleteException(w http.ResponseWriter, r *http.Request) {
	_, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	exID, ok := parseExIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteException(r.Context(), exID); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "exception not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// — Helpers —

func parseExIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "exId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

func parseExceptionInput(w http.ResponseWriter, doctorID int64, req exceptionRequest) (service.ExceptionInput, bool) {
	if req.Date == "" || req.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date and type are required"})
		return service.ExceptionInput{}, false
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date format, expected YYYY-MM-DD"})
		return service.ExceptionInput{}, false
	}

	inp := service.ExceptionInput{
		DoctorID: doctorID,
		Date:     date,
		Type:     model.ExceptionType(req.Type),
		Comment:  req.Comment,
	}

	if req.StartTime != nil {
		t, err := time.Parse("15:04", *req.StartTime)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid start_time format, expected HH:MM"})
			return service.ExceptionInput{}, false
		}
		inp.StartTime = &t
	}
	if req.EndTime != nil {
		t, err := time.Parse("15:04", *req.EndTime)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid end_time format, expected HH:MM"})
			return service.ExceptionInput{}, false
		}
		inp.EndTime = &t
	}

	return inp, true
}

func writeExceptionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "exception not found"})
	case errors.Is(err, apperrors.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "exception already exists for this date"})
	case errors.Is(err, apperrors.ErrInvalidSchedule):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid schedule parameters"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
