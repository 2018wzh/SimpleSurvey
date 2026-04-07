package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/config"
	httpdelivery "github.com/2018wzh/SimpleSurvey/backend/internal/delivery/http"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/auth"
	"go.uber.org/zap"
)

type responseEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newTestRouter() (*config.Config, http.Handler) {
	cfg := &config.Config{
		AppEnv:         "test",
		JWTSecret:      "unit-test-secret",
		AllowedOrigins: []string{"*"},
	}
	h := &httpdelivery.Handler{}
	return cfg, httpdelivery.NewRouter(*cfg, h, zap.NewNop())
}

func makeAccessToken(t *testing.T, secret, uid, role string) string {
	t.Helper()
	token, err := auth.GenerateToken(secret, uid, "tester", role, auth.TokenTypeAccess, time.Hour, "tok-access")
	if err != nil {
		t.Fatalf("generate access token failed: %v", err)
	}
	return token
}

func makeRefreshToken(t *testing.T, secret, uid, role string) string {
	t.Helper()
	token, err := auth.GenerateToken(secret, uid, "tester", role, auth.TokenTypeRefresh, time.Hour, "tok-refresh")
	if err != nil {
		t.Fatalf("generate refresh token failed: %v", err)
	}
	return token
}

func decodeEnvelope(t *testing.T, rr *httptest.ResponseRecorder) responseEnvelope {
	t.Helper()
	var body responseEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v, body=%s", err, rr.Body.String())
	}
	return body
}

func TestRouterHealthEndpoint(t *testing.T) {
	_, router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected health body: %s", rr.Body.String())
	}
}

func TestProtectedRoutesRequireAccessToken(t *testing.T) {
	_, router := newTestRouter()

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/questionnaires", body: ""},
		{method: http.MethodPost, path: "/api/v1/questionnaires", body: `{}`},
		{method: http.MethodGet, path: "/api/v1/questions/q1/versions", body: ""},
		{method: http.MethodPost, path: "/api/v1/question-banks", body: `{}`},
		{method: http.MethodGet, path: "/api/v1/admin/users", body: ""},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s expected 401, got %d, body=%s", tc.method, tc.path, rr.Code, rr.Body.String())
		}
		resp := decodeEnvelope(t, rr)
		if resp.Code != 401 {
			t.Fatalf("%s %s expected business code 401, got %d", tc.method, tc.path, resp.Code)
		}
	}
}

func TestAdminRouteRejectsNonAdminAccessToken(t *testing.T) {
	cfg, router := newTestRouter()
	token := makeAccessToken(t, cfg.JWTSecret, "507f1f77bcf86cd799439011", "user")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeEnvelope(t, rr)
	if resp.Code != 403 {
		t.Fatalf("expected business code 403, got %d", resp.Code)
	}
}

func TestAuthRequiredRejectsRefreshTokenType(t *testing.T) {
	cfg, router := newTestRouter()
	refreshToken := makeRefreshToken(t, cfg.JWTSecret, "507f1f77bcf86cd799439011", "user")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questionnaires", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestOptionalAuthRejectsMalformedAuthorizationHeader(t *testing.T) {
	_, router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/surveys/s1", nil)
	req.Header.Set("Authorization", "Token abc")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeEnvelope(t, rr)
	if resp.Code != 401 {
		t.Fatalf("expected business code 401, got %d", resp.Code)
	}
}
