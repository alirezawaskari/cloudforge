package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/alirezawaskari/cloudforge/internal/store"
	"github.com/alirezawaskari/cloudforge/internal/version"
)

type HealthAPI struct {
	DB    *store.Postgres
	Cache *store.Redis

	// Ready flips to true once startup dependencies are confirmed reachable.
	// The startup probe polls this so slow-starting dependencies don't
	// trip liveness/readiness before the app has had a chance to connect.
	ready atomic.Bool
}

func (h *HealthAPI) SetReady(v bool) {
	h.ready.Store(v)
}

// Livez reports whether the process itself is healthy. It intentionally
// avoids checking downstream dependencies so a slow database doesn't
// cause Kubernetes to kill and restart an otherwise-healthy pod.
func (h *HealthAPI) Livez(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Readyz reports whether the pod should receive traffic: dependencies
// must be reachable.
func (h *HealthAPI) Readyz(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "starting"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{}
	healthy := true

	if err := h.DB.Ping(ctx); err != nil {
		checks["database"] = "down"
		healthy = false
	} else {
		checks["database"] = "up"
	}

	if err := h.Cache.Ping(ctx); err != nil {
		checks["cache"] = "down"
		healthy = false
	} else {
		checks["cache"] = "up"
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]any{"status": statusText(healthy), "checks": checks})
}

// Startupz backs the startup probe: it succeeds only after initial
// dependency connections have completed, giving slow-booting
// dependencies (e.g. Postgres cold start) room without failing
// liveness checks during that window.
func (h *HealthAPI) Startupz(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "starting"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func Version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":    version.Version,
		"commit":     version.Commit,
		"build_date": version.BuildDate,
	})
}

func statusText(healthy bool) string {
	if healthy {
		return "ready"
	}
	return "not_ready"
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
