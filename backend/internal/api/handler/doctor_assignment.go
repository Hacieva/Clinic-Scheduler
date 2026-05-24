package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type DoctorAssignmentHandler struct {
	svc *service.DoctorAssignmentService
}

func NewDoctorAssignmentHandler(svc *service.DoctorAssignmentService) *DoctorAssignmentHandler {
	return &DoctorAssignmentHandler{svc: svc}
}

// ListAssigned returns all services assigned to a doctor via doctor_services junction.
func (h *DoctorAssignmentHandler) ListAssigned(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	svcs, err := h.svc.ListForDoctor(r.Context(), doctorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, svcs)
}

// BulkSet replaces all service assignments for a doctor.
// Body: {"service_ids": [1, 2, 3]}
func (h *DoctorAssignmentHandler) BulkSet(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req struct {
		ServiceIDs []int64 `json:"service_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ServiceIDs == nil {
		req.ServiceIDs = []int64{}
	}
	if err := h.svc.BulkSet(r.Context(), doctorID, req.ServiceIDs); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Assign adds a single service to a doctor's assignment list.
func (h *DoctorAssignmentHandler) Assign(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	serviceID, ok := parseServiceIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Assign(r.Context(), doctorID, serviceID); err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor or service not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Unassign removes a single service from a doctor's assignment list.
func (h *DoctorAssignmentHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	doctorID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	serviceID, ok := parseServiceIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Unassign(r.Context(), doctorID, serviceID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
