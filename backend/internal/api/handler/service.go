package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type ServiceHandler struct {
	svc *service.MedicalServiceService
}

func NewServiceHandler(svc *service.MedicalServiceService) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

type serviceRequest struct {
	DirectionID     int64   `json:"direction_id"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	DurationMinutes int     `json:"duration_minutes"`
	Price           *int64  `json:"price"` // kopecks
}

func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	svcs, err := h.svc.List(r.Context(), doctorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, svcs)
}

func (h *ServiceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	serviceID, ok := parseServiceIDParam(w, r)
	if !ok {
		return
	}
	svc, err := h.svc.GetByID(r.Context(), serviceID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "service not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	// TODO: legacy IDOR check via doctor_id column; remove after bot migrates to doctor_services.
	if svc.DoctorID != nil && *svc.DoctorID != doctorID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "service not found"})
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req serviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.DirectionID == 0 || req.DurationMinutes <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, direction_id and duration_minutes are required"})
		return
	}
	svc, err := h.svc.Create(r.Context(), doctorID, service.ServiceInput{
		DirectionID:     req.DirectionID,
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor not found"})
		case errors.Is(err, apperrors.ErrDirectionMismatch):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "direction does not belong to doctor"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	serviceID, ok := parseServiceIDParam(w, r)
	if !ok {
		return
	}
	var req serviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.DirectionID == 0 || req.DurationMinutes <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, direction_id and duration_minutes are required"})
		return
	}
	svc, err := h.svc.Update(r.Context(), doctorID, serviceID, service.ServiceInput{
		DirectionID:     req.DirectionID,
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		case errors.Is(err, apperrors.ErrDirectionMismatch):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "direction does not belong to doctor"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	_, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	serviceID, ok := parseServiceIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), serviceID); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "service not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseServiceIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "serviceId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return 0, false
	}
	return id, true
}
