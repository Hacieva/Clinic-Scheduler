package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userDTO struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type loginResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         userDTO `json:"user"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	result, err := h.authSvc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound), errors.Is(err, apperrors.ErrUnauthorized):
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		case errors.Is(err, apperrors.ErrInactiveUser):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "account is inactive"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User: userDTO{
			ID:    result.User.ID,
			Email: result.User.Email,
			Role:  string(result.User.Role),
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
