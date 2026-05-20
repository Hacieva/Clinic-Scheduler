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

type DirectionHandler struct {
	svc *service.DirectionService
}

func NewDirectionHandler(svc *service.DirectionService) *DirectionHandler {
	return &DirectionHandler{svc: svc}
}

type directionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (h *DirectionHandler) List(w http.ResponseWriter, r *http.Request) {
	dirs, err := h.svc.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, dirs)
}

func (h *DirectionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	dir, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "direction not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, dir)
}

func (h *DirectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req directionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	dir, err := h.svc.Create(r.Context(), req.Name, req.Description)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, dir)
}

func (h *DirectionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req directionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	dir, err := h.svc.Update(r.Context(), id, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "direction not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, dir)
}

func (h *DirectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	err := h.svc.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "direction not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

