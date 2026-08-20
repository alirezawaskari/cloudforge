// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env  string
	Port string

	DatabaseURL            string
	DBMaxConns             int32
	DBConnTimeout          time.Duration
	DependencyReadyTimeout time.Duration
	RedisAddr              string
	RedisPassword          string
	RedisDB                int
	ShutdownTimeout        time.Duration

	APIKey string

	OTelExporterEndpoint string
	OTelServiceName      string
	OTelEnabled          bool
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:                    getEnv("APP_ENV", "development"),
		Port:                   getEnv("PORT", "8080"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://cloudforge:cloudforge@localhost:5432/cloudforge?sslmode=disable"),
		DBMaxConns:             int32(getEnvInt("DB_MAX_CONNS", 10)),
		DBConnTimeout:          getEnvDuration("DB_CONN_TIMEOUT", 5*time.Second),
		DependencyReadyTimeout: getEnvDuration("DEPENDENCY_READY_TIMEOUT", 120*time.Second),
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		RedisDB:                getEnvInt("REDIS_DB", 0),
		ShutdownTimeout:        getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		OTelExporterEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OTelServiceName:        getEnv("OTEL_SERVICE_NAME", "cloudforge-api"),
		OTelEnabled:            getEnvBool("OTEL_ENABLED", false),
		APIKey:                 getEnv("API_KEY", ""),
	}

	if cfg.Port == "" {
		return nil, fmt.Errorf("PORT must not be empty")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY must be set (used to authenticate writes to /api/v1/items)")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
