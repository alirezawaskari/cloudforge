package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/alirezawaskari/cloudforge/internal/store"
)

const itemCacheTTL = 60 * time.Second

type Item struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type createItemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ItemsAPI struct {
	DB     *store.Postgres
	Cache  *store.Redis
	Logger *slog.Logger
}

func (a *ItemsAPI) Routes(r chi.Router) {
	r.Post("/", a.Create)
	r.Get("/", a.List)
	r.Get("/{id}", a.Get)
	r.Put("/{id}", a.Update)
	r.Delete("/{id}", a.Delete)
}

func (a *ItemsAPI) Create(w http.ResponseWriter, r *http.Request) {
	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	item := Item{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
	}

	const q = `INSERT INTO items (id, name, description) VALUES ($1, $2, $3)
		RETURNING id, name, description, created_at, updated_at`
	row := a.DB.Pool.QueryRow(r.Context(), q, item.ID, item.Name, item.Description)
	if err := row.Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
		a.Logger.Error("create item failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create item")
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (a *ItemsAPI) List(w http.ResponseWriter, r *http.Request) {
	const q = `SELECT id, name, description, created_at, updated_at FROM items ORDER BY created_at DESC LIMIT 100`
	rows, err := a.DB.Pool.Query(r.Context(), q)
	if err != nil {
		a.Logger.Error("list items failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list items")
		return
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt, &it.UpdatedAt); err != nil {
			a.Logger.Error("scan item failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to read items")
			return
		}
		items = append(items, it)
	}

	writeJSON(w, http.StatusOK, items)
}

func (a *ItemsAPI) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	cacheKey := "item:" + id.String()
	if cached, err := a.Cache.Client.Get(r.Context(), cacheKey).Result(); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cached))
		return
	} else if !errors.Is(err, redis.Nil) {
		a.Logger.Warn("redis get failed", "error", err)
	}

	var it Item
	const q = `SELECT id, name, description, created_at, updated_at FROM items WHERE id = $1`
	err = a.DB.Pool.QueryRow(r.Context(), q, id).Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt, &it.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		a.Logger.Error("get item failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get item")
		return
	}

	if payload, err := json.Marshal(it); err == nil {
		if err := a.Cache.Client.Set(r.Context(), cacheKey, payload, itemCacheTTL).Err(); err != nil {
			a.Logger.Warn("redis set failed", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, it)
}

func (a *ItemsAPI) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var it Item
	const q = `UPDATE items SET name = $2, description = $3, updated_at = now()
		WHERE id = $1 RETURNING id, name, description, created_at, updated_at`
	err = a.DB.Pool.QueryRow(r.Context(), q, id, req.Name, req.Description).
		Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt, &it.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		a.Logger.Error("update item failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update item")
		return
	}

	a.invalidateCache(r.Context(), id)
	writeJSON(w, http.StatusOK, it)
}

func (a *ItemsAPI) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	tag, err := a.DB.Pool.Exec(r.Context(), `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		a.Logger.Error("delete item failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete item")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	a.invalidateCache(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}

func (a *ItemsAPI) invalidateCache(ctx context.Context, id uuid.UUID) {
	if err := a.Cache.Client.Del(ctx, "item:"+id.String()).Err(); err != nil {
		a.Logger.Warn("redis del failed", "error", err)
	}
}
