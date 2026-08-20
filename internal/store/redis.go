package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *redis.Client
}

func NewRedis(addr, password string, db int) *Redis {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		Password:    password,
		DB:          db,
		DialTimeout: 5 * time.Second,
		MaxRetries:  3,
	})
	return &Redis{Client: client}
}

func (r *Redis) Ping(ctx context.Context) error {
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}
