package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hacieva/clinic-scheduler/backend/internal/api/middleware"
	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/service"
)

// mockUserRepo implements repository.UserRepository without a real DB.
type mockUserRepo struct {
	user *model.User
	err  error
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) GetByID(_ context.Context, _ int64) (*model.User, error) {
	return m.user, m.err
}

func (m *mockUserRepo) UpdatePassword(_ context.Context, _ int64, _ string) error {
	return nil
}

const (
	testSecret   = "handler-integration-test-secret"
	testPassword = "ValidPass1!"
)

// testPasswordHash is computed once before all tests to avoid bcrypt cost per test.
var testPasswordHash string

func TestMain(m *testing.M) {
	var err error
	testPasswordHash, err = auth.HashPassword(testPassword)
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func activeUser() *model.User {
	return &model.User{
		ID:           42,
		Email:        "admin@clinic.local",
		PasswordHash: testPasswordHash,
		Role:         model.RoleAdmin,
		IsActive:     true,
	}
}

// newTestRouter wires mock repo → real service → real handler → chi router.
func newTestRouter(repo *mockUserRepo) http.Handler {
	svc := service.NewAuthService(repo, testSecret)
	h := NewAuthHandler(svc)
	r := chi.NewRouter()
	r.Post("/api/v1/auth/login", h.Login)
	r.Post("/api/v1/auth/refresh", h.Refresh)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(testSecret))
		r.Post("/api/v1/auth/logout", h.Logout)
		r.Get("/api/v1/auth/me", h.Me)
		r.Post("/api/v1/auth/change-password", h.ChangePassword)
	})
	return r
}

func postLogin(router http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func bearerReq(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestLogin_Success(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})
	rr := postLogin(router, `{"email":"admin@clinic.local","password":"ValidPass1!"}`)

	require.Equal(t, http.StatusOK, rr.Code)

	rawBody := rr.Body.Bytes()

	var resp loginResponse
	require.NoError(t, json.Unmarshal(rawBody, &resp))

	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, int64(42), resp.User.ID)
	assert.Equal(t, "admin@clinic.local", resp.User.Email)
	assert.Equal(t, "admin", resp.User.Role)

	// Access token must be a valid JWT with correct claims.
	claims, err := auth.ValidateToken(resp.AccessToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, model.RoleAdmin, claims.Role)

	// Password hash must never appear in any form in the response.
	bodyStr := string(rawBody)
	assert.NotContains(t, bodyStr, "password_hash")
	assert.NotContains(t, bodyStr, testPasswordHash)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})
	rr := postLogin(router, `{"email":"admin@clinic.local","password":"WrongPassword!"}`)

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "invalid credentials", resp["error"])
}

func TestLogin_InactiveUser(t *testing.T) {
	user := activeUser()
	user.IsActive = false
	router := newTestRouter(&mockUserRepo{user: user})
	rr := postLogin(router, `{"email":"admin@clinic.local","password":"ValidPass1!"}`)

	require.Equal(t, http.StatusForbidden, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "account is inactive", resp["error"])
}

func TestLogin_MalformedJSON(t *testing.T) {
	router := newTestRouter(&mockUserRepo{})
	rr := postLogin(router, `{not valid json`)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestLogin_MissingFields(t *testing.T) {
	router := newTestRouter(&mockUserRepo{})

	cases := []struct {
		name string
		body string
	}{
		{"missing_password", `{"email":"admin@clinic.local"}`},
		{"missing_email", `{"password":"ValidPass1!"}`},
		{"both_empty_strings", `{"email":"","password":""}`},
		{"empty_object", `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := postLogin(router, tc.body)
			require.Equal(t, http.StatusBadRequest, rr.Code)

			var resp map[string]string
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, "email and password are required", resp["error"])
		})
	}
}

func TestLogin_WrongMethod(t *testing.T) {
	router := newTestRouter(&mockUserRepo{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// — Refresh —

func TestRefresh_Success(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	refreshToken, err := auth.GenerateRefreshToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"`+refreshToken+`"}`, "")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp loginResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, int64(42), resp.User.ID)

	accessClaims, err := auth.ValidateToken(resp.AccessToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "access", accessClaims.TokenType)
	assert.Equal(t, int64(42), accessClaims.UserID)

	refreshClaims, err := auth.ValidateToken(resp.RefreshToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "refresh", refreshClaims.TokenType)
}

func TestRefresh_InvalidToken(t *testing.T) {
	router := newTestRouter(&mockUserRepo{})
	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"not.a.valid.token"}`, "")

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "invalid or expired token", resp["error"])
}

func TestRefresh_AccessTokenRejected(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	accessToken, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"`+accessToken+`"}`, "")

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "invalid or expired token", resp["error"])
}

func TestRefresh_MissingField(t *testing.T) {
	router := newTestRouter(&mockUserRepo{})
	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/refresh", `{}`, "")

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "refresh_token is required", resp["error"])
}

func TestRefresh_InactiveUser(t *testing.T) {
	user := activeUser()
	user.IsActive = false
	router := newTestRouter(&mockUserRepo{user: user})

	refreshToken, err := auth.GenerateRefreshToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"`+refreshToken+`"}`, "")

	require.Equal(t, http.StatusForbidden, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "account is inactive", resp["error"])
}

// — Logout —

func TestLogout_Success(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	accessToken, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/logout", "", accessToken)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "logged out", resp["message"])
}

func TestLogout_NoToken(t *testing.T) {
	router := newTestRouter(&mockUserRepo{})
	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/logout", "", "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestLogout_WithRefreshToken(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	refreshToken, err := auth.GenerateRefreshToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/logout", "", refreshToken)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — Me —

func TestMe_Success(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	accessToken, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodGet, "/api/v1/auth/me", "", accessToken)

	require.Equal(t, http.StatusOK, rr.Code)

	rawBody := rr.Body.Bytes()

	var resp userDTO
	require.NoError(t, json.Unmarshal(rawBody, &resp))
	assert.Equal(t, int64(42), resp.ID)
	assert.Equal(t, "admin@clinic.local", resp.Email)
	assert.Equal(t, "admin", resp.Role)

	bodyStr := string(rawBody)
	assert.NotContains(t, bodyStr, "password_hash")
	assert.NotContains(t, bodyStr, testPasswordHash)
}

func TestMe_NoToken(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})
	rr := bearerReq(router, http.MethodGet, "/api/v1/auth/me", "", "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestMe_WithRefreshToken(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	refreshToken, err := auth.GenerateRefreshToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodGet, "/api/v1/auth/me", "", refreshToken)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — ChangePassword —

func TestChangePassword_Success(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	accessToken, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/change-password",
		`{"current_password":"ValidPass1!","new_password":"NewSecure99!"}`, accessToken)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	accessToken, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/change-password",
		`{"current_password":"WrongPassword!","new_password":"NewSecure99!"}`, accessToken)

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "invalid current password", resp["error"])
}

func TestChangePassword_WeakNew(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	accessToken, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/change-password",
		`{"current_password":"ValidPass1!","new_password":"short"}`, accessToken)

	require.Equal(t, http.StatusUnprocessableEntity, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "password must be at least 8 characters", resp["error"])
}

func TestChangePassword_MissingFields(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})

	accessToken, err := auth.GenerateAccessToken(42, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	cases := []struct {
		name string
		body string
	}{
		{"missing_new", `{"current_password":"ValidPass1!"}`},
		{"missing_current", `{"new_password":"NewSecure99!"}`},
		{"both_empty", `{"current_password":"","new_password":""}`},
		{"empty_object", `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := bearerReq(router, http.MethodPost, "/api/v1/auth/change-password",
				tc.body, accessToken)
			require.Equal(t, http.StatusBadRequest, rr.Code)

			var resp map[string]string
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, "current_password and new_password are required", resp["error"])
		})
	}
}

func TestChangePassword_NoToken(t *testing.T) {
	router := newTestRouter(&mockUserRepo{user: activeUser()})
	rr := bearerReq(router, http.MethodPost, "/api/v1/auth/change-password",
		`{"current_password":"ValidPass1!","new_password":"NewSecure99!"}`, "")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
