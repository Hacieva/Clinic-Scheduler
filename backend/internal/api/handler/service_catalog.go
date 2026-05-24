package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type ServiceCatalogHandler struct {
	svc *service.ServiceCatalogService
}

func NewServiceCatalogHandler(svc *service.ServiceCatalogService) *ServiceCatalogHandler {
	return &ServiceCatalogHandler{svc: svc}
}

type catalogServiceRequest struct {
	DirectionID     int64   `json:"direction_id"`
	Category        *string `json:"category"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	DurationMinutes int     `json:"duration_minutes"`
	Price           *int64  `json:"price"` // kopecks
}

// ListAll lists all catalog services. Pass ?active_only=false to include inactive.
func (h *ServiceCatalogHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") != "false"
	svcs, err := h.svc.ListAll(r.Context(), activeOnly)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, svcs)
}

func (h *ServiceCatalogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req catalogServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.DirectionID == 0 || req.DurationMinutes <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, direction_id and duration_minutes are required"})
		return
	}
	svc, err := h.svc.Create(r.Context(), service.CatalogServiceInput{
		DirectionID:     req.DirectionID,
		Category:        req.Category,
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (h *ServiceCatalogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req catalogServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.DirectionID == 0 || req.DurationMinutes <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, direction_id and duration_minutes are required"})
		return
	}
	svc, err := h.svc.Update(r.Context(), id, service.CatalogServiceInput{
		DirectionID:     req.DirectionID,
		Category:        req.Category,
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "service not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ServiceCatalogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "service not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
