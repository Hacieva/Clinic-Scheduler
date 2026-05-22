package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Hacieva/clinic-scheduler/backend/internal/api/middleware"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type AppointmentHandler struct {
	svc *service.AppointmentService
}

func NewAppointmentHandler(svc *service.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{svc: svc}
}

// createAppointmentRequest is shared by admin and bot create endpoints.
// TelegramID/Username are set only by the bot; Source is set by the handler.
type createAppointmentRequest struct {
	PatientTelegramID       *int64  `json:"patient_telegram_id"`
	PatientTelegramUsername *string `json:"patient_telegram_username"`
	PatientName             string  `json:"patient_name"`
	PatientPhone            string  `json:"patient_phone"`
	DoctorID                int64   `json:"doctor_id"`
	ServiceID               int64   `json:"service_id"`
	StartAt                 string  `json:"start_at"` // RFC3339
	PatientComment          *string `json:"patient_comment"`
}

// appointmentDoctorView omits sensitive patient fields for doctor-facing responses.
// Trimming happens here in the handler layer, not in service or repository.
type appointmentDoctorView struct {
	model.Appointment
	PatientName    string `json:"patient_name"`
	DoctorFullName string `json:"doctor_full_name"`
	ServiceName    string `json:"service_name"`
}

func toDoctorView(d repository.AppointmentDetail) appointmentDoctorView {
	return appointmentDoctorView{
		Appointment:    d.Appointment,
		PatientName:    d.PatientName,
		DoctorFullName: d.DoctorFullName,
		ServiceName:    d.ServiceName,
	}
}

// — Admin endpoints (JWT + admin role) —

func (h *AppointmentHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	req, startAt, ok := parseCreateRequest(w, r)
	if !ok {
		return
	}
	uid := claims.UserID
	appt, err := h.svc.Create(r.Context(), service.CreateAppointmentInput{
		PatientName:     req.PatientName,
		PatientPhone:    req.PatientPhone,
		DoctorID:        req.DoctorID,
		ServiceID:       req.ServiceID,
		StartAt:         startAt,
		Source:          model.SourceAdminPanel,
		PatientComment:  req.PatientComment,
		CreatedByUserID: &uid,
	})
	if err != nil {
		writeAppointmentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, appt)
}

func (h *AppointmentHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseAppointmentFilter(w, r)
	if !ok {
		return
	}
	list, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *AppointmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	detail, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "appointment not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *AppointmentHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid := claims.UserID
	if err := h.svc.Confirm(r.Context(), id, &uid); err != nil {
		writeAppointmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AppointmentHandler) AdminCancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req struct {
		Comment *string `json:"comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	uid := claims.UserID
	if err := h.svc.CancelByAdmin(r.Context(), id, &uid, req.Comment); err != nil {
		writeAppointmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AppointmentHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid := claims.UserID
	if err := h.svc.Complete(r.Context(), id, &uid); err != nil {
		writeAppointmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AppointmentHandler) MarkNoShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid := claims.UserID
	if err := h.svc.MarkNoShow(r.Context(), id, &uid); err != nil {
		writeAppointmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// — Doctor endpoints (JWT + doctor role) —

// DoctorList returns the authenticated doctor's own appointments with privacy trimming.
// The doctor_id is resolved from the JWT claims — query param doctor_id is ignored.
func (h *AppointmentHandler) DoctorList(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	doctorID, err := h.svc.GetDoctorIDByUserID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor profile not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	filter, ok := parseAppointmentFilter(w, r)
	if !ok {
		return
	}
	// Force doctor_id from JWT claims — query param override is silently ignored.
	filter.DoctorID = &doctorID

	list, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	views := make([]appointmentDoctorView, 0, len(list))
	for _, d := range list {
		views = append(views, toDoctorView(d))
	}
	writeJSON(w, http.StatusOK, views)
}

// DoctorGetByID returns a single appointment, only if it belongs to the authenticated doctor.
func (h *AppointmentHandler) DoctorGetByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	doctorID, err := h.svc.GetDoctorIDByUserID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doctor profile not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	detail, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "appointment not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if detail.DoctorID != doctorID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	writeJSON(w, http.StatusOK, toDoctorView(*detail))
}

// — Bot endpoints (X-Bot-Token auth) —

func (h *AppointmentHandler) BotCreate(w http.ResponseWriter, r *http.Request) {
	req, startAt, ok := parseCreateRequest(w, r)
	if !ok {
		return
	}
	appt, err := h.svc.Create(r.Context(), service.CreateAppointmentInput{
		PatientTelegramID:       req.PatientTelegramID,
		PatientTelegramUsername: req.PatientTelegramUsername,
		PatientName:             req.PatientName,
		PatientPhone:            req.PatientPhone,
		DoctorID:                req.DoctorID,
		ServiceID:               req.ServiceID,
		StartAt:                 startAt,
		Source:                  model.SourceTelegramBot,
		PatientComment:          req.PatientComment,
	})
	if err != nil {
		writeAppointmentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, appt)
}

func (h *AppointmentHandler) BotCancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.CancelByPatient(r.Context(), id); err != nil {
		writeAppointmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// — Helpers —

func parseCreateRequest(w http.ResponseWriter, r *http.Request) (createAppointmentRequest, time.Time, bool) {
	var req createAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return req, time.Time{}, false
	}
	if req.PatientName == "" || req.PatientPhone == "" || req.DoctorID <= 0 || req.ServiceID <= 0 || req.StartAt == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "patient_name, patient_phone, doctor_id, service_id, start_at are required"})
		return req, time.Time{}, false
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid start_at, expected RFC3339"})
		return req, time.Time{}, false
	}
	return req, startAt, true
}

func parseAppointmentFilter(w http.ResponseWriter, r *http.Request) (repository.AppointmentFilter, bool) {
	q := r.URL.Query()
	filter := repository.AppointmentFilter{}

	if v := q.Get("doctor_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid doctor_id"})
			return filter, false
		}
		filter.DoctorID = &id
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
		s := model.AppointmentStatus(v)
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
		if err != nil || n < 1 || n > 200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be between 1 and 200"})
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

func writeAppointmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, apperrors.ErrSlotTaken):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "time slot already taken"})
	case errors.Is(err, apperrors.ErrDoctorInactive):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "doctor is inactive"})
	case errors.Is(err, apperrors.ErrDirectionMismatch):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "service does not belong to this doctor"})
	case errors.Is(err, apperrors.ErrOutsideHours):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "appointment time is outside working hours"})
	case errors.Is(err, apperrors.ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid status transition"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
