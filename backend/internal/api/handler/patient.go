package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type PatientHandler struct {
	svc *service.PatientService
}

func NewPatientHandler(svc *service.PatientService) *PatientHandler {
	return &PatientHandler{svc: svc}
}

// GET /patients?search=&limit=&offset=
func (h *PatientHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := repository.PatientFilter{}

	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = &s
	}
	if s := r.URL.Query().Get("source"); s != "" {
		filter.Source = &s
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writePatientError(w, apperrors.ErrInvalidInput)
			return
		}
		filter.Limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writePatientError(w, apperrors.ErrInvalidInput)
			return
		}
		filter.Offset = n
	}

	patients, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writePatientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, patients)
}

// GET /patients/{id}
func (h *PatientHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writePatientError(w, apperrors.ErrInvalidInput)
		return
	}

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writePatientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type createPatientRequest struct {
	FullName    string  `json:"full_name"`
	Phone       string  `json:"phone"`
	Email       *string `json:"email"`
	DateOfBirth *string `json:"date_of_birth"` // "YYYY-MM-DD"
	Comment     *string `json:"comment"`
}

// POST /patients
func (h *PatientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writePatientError(w, apperrors.ErrInvalidInput)
		return
	}
	if req.FullName == "" || req.Phone == "" {
		writePatientError(w, apperrors.ErrInvalidInput)
		return
	}

	input := repository.CreatePatientInput{
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Comment:  req.Comment,
		Source:   "admin_panel",
	}
	if req.DateOfBirth != nil {
		t, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			writePatientError(w, apperrors.ErrInvalidInput)
			return
		}
		input.DateOfBirth = &t
	}

	p, err := h.svc.Create(r.Context(), input)
	if err != nil {
		writePatientError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

type updatePatientRequest struct {
	FullName    *string `json:"full_name"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	DateOfBirth *string `json:"date_of_birth"` // "YYYY-MM-DD"
	Comment     *string `json:"comment"`
}

// PATCH /patients/{id}
func (h *PatientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writePatientError(w, apperrors.ErrInvalidInput)
		return
	}

	var req updatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writePatientError(w, apperrors.ErrInvalidInput)
		return
	}

	input := repository.UpdatePatientInput{
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Comment:  req.Comment,
	}
	if req.DateOfBirth != nil {
		t, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			writePatientError(w, apperrors.ErrInvalidInput)
			return
		}
		input.DateOfBirth = &t
	}

	p, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		writePatientError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func writePatientError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "patient not found"})
	case errors.Is(err, apperrors.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
