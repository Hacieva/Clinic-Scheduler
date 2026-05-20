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
	return r
}

func postLogin(router http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
