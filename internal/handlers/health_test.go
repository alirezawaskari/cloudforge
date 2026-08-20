package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivezAlwaysOK(t *testing.T) {
	h := &HealthAPI{}
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	rec := httptest.NewRecorder()

	h.Livez(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReadyzNotReadyBeforeStartup(t *testing.T) {
	h := &HealthAPI{}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before ready, got %d", rec.Code)
	}
}

func TestStartupzTracksReadyFlag(t *testing.T) {
	h := &HealthAPI{}
	req := httptest.NewRequest(http.MethodGet, "/startupz", nil)

	rec := httptest.NewRecorder()
	h.Startupz(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before ready, got %d", rec.Code)
	}

	h.SetReady(true)

	rec = httptest.NewRecorder()
	h.Startupz(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 after ready, got %d", rec.Code)
	}
}

func TestVersionEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	Version(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected json content type, got %q", ct)
	}
}
