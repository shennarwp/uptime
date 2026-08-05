package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAPIToken(t *testing.T) {
	t.Setenv("UPTIME_API_TOKEN", "sekrit")

	next := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"valid token", "Bearer sekrit", http.StatusOK},
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic sekrit", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/target/1", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			RequireAPIToken(next)(rec, req)

			if rec.Code != tc.want {
				t.Errorf("expected status %d, got %d", tc.want, rec.Code)
			}
		})
	}
}

func TestRequireAPIToken_UnsetEnvDenies(t *testing.T) {
	t.Setenv("UPTIME_API_TOKEN", "")

	req := httptest.NewRequest("PUT", "/api/target/1", nil)
	rec := httptest.NewRecorder()

	RequireAPIToken(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 when env unset, got %d", rec.Code)
	}
}

func TestRequireAPIToken_UnsetEnvDeniesEvenWithHeader(t *testing.T) {
	t.Setenv("UPTIME_API_TOKEN", "")

	req := httptest.NewRequest("PUT", "/api/target/1", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()

	RequireAPIToken(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 when env unset, got %d", rec.Code)
	}
}

func TestVerifyToken(t *testing.T) {
	t.Setenv("UPTIME_API_TOKEN", "sekrit")

	h := &TargetHandler{}

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"valid token", "Bearer sekrit", http.StatusOK},
		{"no header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer nope", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/verify", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			h.VerifyToken(rec, req)

			if rec.Code != tc.want {
				t.Errorf("expected status %d, got %d", tc.want, rec.Code)
			}
		})
	}
}

func TestVerifyToken_UnsetEnvDenies(t *testing.T) {
	t.Setenv("UPTIME_API_TOKEN", "")

	h := &TargetHandler{}
	req := httptest.NewRequest("POST", "/api/auth/verify", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()

	h.VerifyToken(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 when env unset, got %d", rec.Code)
	}
}
