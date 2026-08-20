package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAPIKeyRejectsMissingKey(t *testing.T) {
	h := RequireAPIKey("secret")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAPIKeyRejectsWrongKey(t *testing.T) {
	h := RequireAPIKey("secret")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", nil)
	req.Header.Set("X-API-Key", "wrong")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAPIKeyAcceptsCorrectKey(t *testing.T) {
	h := RequireAPIKey("secret")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", nil)
	req.Header.Set("X-API-Key", "secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAPIKeyRejectsEverythingWhenUnconfigured(t *testing.T) {
	h := RequireAPIKey("")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", nil)
	req.Header.Set("X-API-Key", "")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no key is configured, got %d", rec.Code)
	}
}
