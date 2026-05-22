package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

const testSecret = "test-middleware-secret"

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func tokenFor(t *testing.T, userID int64, role model.UserRole) string {
	t.Helper()
	tok, err := auth.GenerateAccessToken(userID, role, nil, testSecret)
	require.NoError(t, err)
	return tok
}

func bearerReq(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

// — Authenticate —

func TestAuthenticate_ValidToken(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleAdmin)
	h := Authenticate(testSecret)(okHandler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthenticate_NoToken(t *testing.T) {
	h := Authenticate(testSecret)(okHandler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(""))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	h := Authenticate(testSecret)(okHandler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq("not.a.valid.token"))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticate_RefreshTokenRejected(t *testing.T) {
	tok, err := auth.GenerateRefreshToken(1, model.RoleAdmin, nil, testSecret)
	require.NoError(t, err)

	h := Authenticate(testSecret)(okHandler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticate_WrongSecret(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleAdmin)
	h := Authenticate("wrong-secret")(okHandler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// — RequireRole: admin-only —

func TestRequireRole_AdminOnly_AdminPasses(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleAdmin)
	h := Authenticate(testSecret)(RequireRole("admin")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_AdminOnly_OwnerForbidden(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleOwner)
	h := Authenticate(testSecret)(RequireRole("admin")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireRole_AdminOnly_DoctorForbidden(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleDoctor)
	h := Authenticate(testSecret)(RequireRole("admin")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — RequireRole: owner-only —

func TestRequireRole_OwnerOnly_OwnerPasses(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleOwner)
	h := Authenticate(testSecret)(RequireRole("owner")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_OwnerOnly_AdminForbidden(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleAdmin)
	h := Authenticate(testSecret)(RequireRole("owner")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireRole_OwnerOnly_DoctorForbidden(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleDoctor)
	h := Authenticate(testSecret)(RequireRole("owner")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — RequireRole: owner + admin —

func TestRequireRole_OwnerAdmin_OwnerPasses(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleOwner)
	h := Authenticate(testSecret)(RequireRole("owner", "admin")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_OwnerAdmin_AdminPasses(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleAdmin)
	h := Authenticate(testSecret)(RequireRole("owner", "admin")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_OwnerAdmin_DoctorForbidden(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleDoctor)
	h := Authenticate(testSecret)(RequireRole("owner", "admin")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — RequireRole: doctor-only —

func TestRequireRole_DoctorOnly_DoctorPasses(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleDoctor)
	h := Authenticate(testSecret)(RequireRole("doctor")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_DoctorOnly_AdminForbidden(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleAdmin)
	h := Authenticate(testSecret)(RequireRole("doctor")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireRole_DoctorOnly_OwnerForbidden(t *testing.T) {
	tok := tokenFor(t, 1, model.RoleOwner)
	h := Authenticate(testSecret)(RequireRole("doctor")(okHandler()))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, bearerReq(tok))
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// — RequireRole without Authenticate (no claims in context) —

func TestRequireRole_NoClaimsInContext_Unauthorized(t *testing.T) {
	h := RequireRole("admin")(okHandler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
