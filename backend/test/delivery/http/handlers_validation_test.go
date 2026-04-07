package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthEndpointValidationErrors(t *testing.T) {
	_, router := newTestRouter()

	cases := []struct {
		path string
		body string
	}{
		{path: "/api/v1/auth/register", body: `{}`},
		{path: "/api/v1/auth/login", body: `{}`},
		{path: "/api/v1/auth/refresh", body: `{}`},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("POST %s expected 400, got %d, body=%s", tc.path, rr.Code, rr.Body.String())
		}
		resp := decodeEnvelope(t, rr)
		if resp.Code != 400 {
			t.Fatalf("POST %s expected business code 400, got %d", tc.path, resp.Code)
		}
	}
}

func TestProtectedEndpointValidationWithValidToken(t *testing.T) {
	cfg, router := newTestRouter()
	token := makeAccessToken(t, cfg.JWTSecret, "507f1f77bcf86cd799439011", "user")

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/v1/questions", body: `{"questionKey":"not-empty"}`},
		{method: http.MethodPost, path: "/api/v1/questions/q1/versions", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/questions/q1/restore", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/question-banks", body: `{}`},
		{method: http.MethodPatch, path: "/api/v1/question-banks/b1", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/question-banks/b1/items", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/question-banks/b1/shares", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/questionnaires", body: `{}`},
		{method: http.MethodPatch, path: "/api/v1/questionnaires/q1/status", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/questionnaires/q1/reports/crosstab", body: `{}`},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s %s expected 400, got %d, body=%s", tc.method, tc.path, rr.Code, rr.Body.String())
		}
		resp := decodeEnvelope(t, rr)
		if resp.Code != 400 {
			t.Fatalf("%s %s expected business code 400, got %d", tc.method, tc.path, resp.Code)
		}
	}
}

func TestSurveySubmitValidationError(t *testing.T) {
	_, router := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/surveys/s1/responses", strings.NewReader(`{"isAnonymous":true}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeEnvelope(t, rr)
	if resp.Code != 400 {
		t.Fatalf("expected business code 400, got %d", resp.Code)
	}
}

func TestCreateCrossTabValidationErrorForInvalidDateFilter(t *testing.T) {
	cfg, router := newTestRouter()
	token := makeAccessToken(t, cfg.JWTSecret, "507f1f77bcf86cd799439011", "user")

	body := `{"rowQuestionId":"q1","colQuestionId":"q2","filters":{"dateRange":{"start":"bad-date"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questionnaires/q1/reports/crosstab", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeEnvelope(t, rr)
	if resp.Code != 400 {
		t.Fatalf("expected business code 400, got %d", resp.Code)
	}
}
