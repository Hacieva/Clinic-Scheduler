package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type DoctorHandler struct {
	svc *service.DoctorService
}

func NewDoctorHandler(svc *service.DoctorService) *DoctorHandler {
	return &DoctorHandler{svc: svc}
}

type doctorRequest struct {
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	MiddleName    *string `json:"middle_name"`
	Cabinet       *string `json:"cabinet"`
	BranchAddress *string `json:"branch_address"`
	Description   *string `json:"description"`
	PhotoURL      *string `json:"photo_url"`
}

type createAccountRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type setDirectionsRequest struct {
	DirectionIDs []int64 `json:"direction_ids"`
}

func (h *DoctorHandler) List(w http.ResponseWriter, r *http.Request) {
	var directionID *int64
	if v := r.URL.Query().Get("direction_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid direction_id"})
			return
		}
		directionID = &id
	}
	docs, err := h.svc.List(r.Context(), directionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

func (h *DoctorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	doc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *DoctorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req doctorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.FirstName == "" || req.LastName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "first_name and last_name are required"})
		return
	}
	doc, err := h.svc.Create(r.Context(), service.DoctorInput{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		MiddleName:    req.MiddleName,
		Cabinet:       req.Cabinet,
		BranchAddress: req.BranchAddress,
		Description:   req.Description,
		PhotoURL:      req.PhotoURL,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *DoctorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req doctorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.FirstName == "" || req.LastName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "first_name and last_name are required"})
		return
	}
	doc, err := h.svc.Update(r.Context(), id, service.DoctorInput{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		MiddleName:    req.MiddleName,
		Cabinet:       req.Cabinet,
		BranchAddress: req.BranchAddress,
		Description:   req.Description,
		PhotoURL:      req.PhotoURL,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *DoctorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DoctorHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}
	doc, err := h.svc.CreateAccount(r.Context(), id, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor not found"})
		case errors.Is(err, apperrors.ErrAccountExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "doctor already has an account"})
		case errors.Is(err, apperrors.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already taken"})
		case errors.Is(err, auth.ErrWeakPassword):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *DoctorHandler) SetDirections(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var req setDirectionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.DirectionIDs == nil {
		req.DirectionIDs = []int64{}
	}
	if err := h.svc.SetDirections(r.Context(), id, req.DirectionIDs); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
