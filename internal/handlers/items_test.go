package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/alirezawaskari/cloudforge/internal/store"
)

// newTestItemsAPI wires an ItemsAPI against the real Postgres/Redis started
// by docker-compose (local) or the CI service containers (see
// .github/workflows/ci.yml), matching how the app actually runs -- these
// are integration tests, not a mocked stand-in for the database.
func newTestItemsAPI(t *testing.T) *ItemsAPI {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://cloudforge:cloudforge@localhost:5432/cloudforge?sslmode=disable"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := store.NewPostgres(ctx, dbURL, 5)
	if err != nil {
		t.Skipf("skipping: postgres unavailable: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		t.Skipf("skipping: postgres unreachable: %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cache := store.NewRedis(redisAddr, "", 0)
	if err := cache.Ping(ctx); err != nil {
		t.Skipf("skipping: redis unreachable: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Pool.Exec(context.Background(), "DELETE FROM items")
		db.Close()
		cache.Close()
	})

	return &ItemsAPI{
		DB:     db,
		Cache:  cache,
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}

func newTestRouter(items *ItemsAPI) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1/items", items.Routes)
	return r
}

func doRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestItemsCreateGetUpdateDeleteLifecycle(t *testing.T) {
	api := newTestItemsAPI(t)
	router := newTestRouter(api)

	rec := doRequest(t, router, http.MethodPost, "/api/v1/items", createItemRequest{Name: "Widget", Description: "a widget"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created Item
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created item: %v", err)
	}
	if created.Name != "Widget" {
		t.Fatalf("expected name Widget, got %q", created.Name)
	}

	path := fmt.Sprintf("/api/v1/items/%s", created.ID)

	// First read misses the cache and populates it.
	rec = doRequest(t, router, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Cache") == "HIT" {
		t.Fatal("expected a cache miss on first read")
	}

	// Second read should be served from cache.
	rec = doRequest(t, router, http.MethodGet, path, nil)
	if rec.Header().Get("X-Cache") != "HIT" {
		t.Fatal("expected a cache hit on second read")
	}

	rec = doRequest(t, router, http.MethodPut, path, createItemRequest{Name: "Widget v2", Description: "updated"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Update must invalidate the cache -- the next read reflects the change
	// rather than serving the stale cached copy.
	rec = doRequest(t, router, http.MethodGet, path, nil)
	var updated Item
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated item: %v", err)
	}
	if updated.Name != "Widget v2" {
		t.Fatalf("expected updated name after cache invalidation, got %q", updated.Name)
	}

	rec = doRequest(t, router, http.MethodDelete, path, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", rec.Code)
	}

	rec = doRequest(t, router, http.MethodGet, path, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", rec.Code)
	}
}

func TestItemsCreateRejectsMissingName(t *testing.T) {
	api := newTestItemsAPI(t)
	router := newTestRouter(api)

	rec := doRequest(t, router, http.MethodPost, "/api/v1/items", createItemRequest{Description: "no name"})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d", rec.Code)
	}
}

func TestItemsGetUnknownIDReturnsNotFound(t *testing.T) {
	api := newTestItemsAPI(t)
	router := newTestRouter(api)

	rec := doRequest(t, router, http.MethodGet, "/api/v1/items/00000000-0000-0000-0000-000000000000", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestItemsGetInvalidIDReturnsBadRequest(t *testing.T) {
	api := newTestItemsAPI(t)
	router := newTestRouter(api)

	rec := doRequest(t, router, http.MethodGet, "/api/v1/items/not-a-uuid", nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestItemsListReturnsCreatedItems(t *testing.T) {
	api := newTestItemsAPI(t)
	router := newTestRouter(api)

	doRequest(t, router, http.MethodPost, "/api/v1/items", createItemRequest{Name: "A"})
	doRequest(t, router, http.MethodPost, "/api/v1/items", createItemRequest{Name: "B"})

	rec := doRequest(t, router, http.MethodGet, "/api/v1/items", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var items []Item
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestItemsDeleteUnknownIDReturnsNotFound(t *testing.T) {
	api := newTestItemsAPI(t)
	router := newTestRouter(api)

	rec := doRequest(t, router, http.MethodDelete, "/api/v1/items/00000000-0000-0000-0000-000000000000", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
