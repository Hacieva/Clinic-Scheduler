package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/api/middleware"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type VisitHandler struct {
	svc *service.VisitService
}

func NewVisitHandler(svc *service.VisitService) *VisitHandler {
	return &VisitHandler{svc: svc}
}

// POST /visits — walk-in patient registration.
// Creates both a Visit (walk_in, in_progress) and an Appointment (walk_in, arrived).
func (h *VisitHandler) RegisterWalkIn(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		PatientName    string  `json:"patient_name"`
		PatientPhone   string  `json:"patient_phone"`
		DoctorID       int64   `json:"doctor_id"`
		ServiceID      int64   `json:"service_id"`
		BranchID       int64   `json:"branch_id"`
		PatientComment *string `json:"patient_comment"`
		Comment        *string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.PatientName == "" || req.PatientPhone == "" || req.DoctorID <= 0 || req.ServiceID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "patient_name, patient_phone, doctor_id, service_id are required"})
		return
	}

	uid := claims.UserID
	visit, appt, err := h.svc.RegisterWalkIn(r.Context(), service.RegisterWalkInInput{
		PatientName:     req.PatientName,
		PatientPhone:    req.PatientPhone,
		DoctorID:        req.DoctorID,
		ServiceID:       req.ServiceID,
		BranchID:        req.BranchID,
		Source:          model.SourceAdminPanel,
		PatientComment:  req.PatientComment,
		Comment:         req.Comment,
		CreatedByUserID: &uid,
	})
	if err != nil {
		writeVisitError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"visit": visit, "appointment": appt})
}

// GET /visits — list visits with optional filters.
func (h *VisitHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseVisitFilter(w, r)
	if !ok {
		return
	}
	visits, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, visits)
}

// GET /visits/{id} — single visit by ID.
func (h *VisitHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	visit, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "visit not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, visit)
}

func parseVisitFilter(w http.ResponseWriter, r *http.Request) (repository.VisitFilter, bool) {
	q := r.URL.Query()
	filter := repository.VisitFilter{}

	if v := q.Get("patient_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid patient_id"})
			return filter, false
		}
		filter.PatientID = &id
	}

	if v := q.Get("branch_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid branch_id"})
			return filter, false
		}
		filter.BranchID = &id
	}

	if v := q.Get("status"); v != "" {
		s := model.VisitStatus(v)
		filter.Status = &s
	}

	if v := q.Get("date_from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date_from, expected YYYY-MM-DD"})
			return filter, false
		}
		filter.DateFrom = &t
	}

	if v := q.Get("date_to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date_to, expected YYYY-MM-DD"})
			return filter, false
		}
		filter.DateTo = &t
	}

	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be between 1 and 100"})
			return filter, false
		}
		filter.Limit = n
	}

	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid offset"})
			return filter, false
		}
		filter.Offset = n
	}

	return filter, true
}

func writeVisitError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, apperrors.ErrDoctorInactive):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "doctor is inactive"})
	case errors.Is(err, apperrors.ErrDirectionMismatch):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "service does not belong to this doctor"})
	case errors.Is(err, apperrors.ErrInvalidBookingMode):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "booking mode does not allow walk-in appointments"})
	case errors.Is(err, apperrors.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
