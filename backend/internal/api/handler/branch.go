package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type BranchHandler struct {
	svc *service.BranchService
}

func NewBranchHandler(svc *service.BranchService) *BranchHandler {
	return &BranchHandler{svc: svc}
}

type branchRequest struct {
	Name    string  `json:"name"`
	Address *string `json:"address"`
	Phone   *string `json:"phone"`
}

func (h *BranchHandler) List(w http.ResponseWriter, r *http.Request) {
	branches, err := h.svc.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, branches)
}

func (h *BranchHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	branch, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeBranchError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, branch)
}

func (h *BranchHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req branchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	branch, err := h.svc.Create(r.Context(), service.BranchInput{
		Name:    req.Name,
		Address: req.Address,
		Phone:   req.Phone,
	})
	if err != nil {
		writeBranchError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, branch)
}

func (h *BranchHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req branchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	branch, err := h.svc.Update(r.Context(), id, service.BranchInput{
		Name:    req.Name,
		Address: req.Address,
		Phone:   req.Phone,
	})
	if err != nil {
		writeBranchError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, branch)
}

func (h *BranchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Deactivate(r.Context(), id); err != nil {
		writeBranchError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeBranchError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "branch not found"})
	case errors.Is(err, apperrors.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
	case errors.Is(err, apperrors.ErrBranchHasActiveDoctors):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "branch has active doctors"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
