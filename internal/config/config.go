package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            int
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	HeartbeatTTL    time.Duration
	CleanupInterval time.Duration
	RequestTimeout  time.Duration
}

func Load() *Config {
	return &Config{
		Port:            getEnvInt("PORT", 8080),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         getEnvInt("REDIS_DB", 0),
		HeartbeatTTL:    getEnvDuration("HEARTBEAT_TTL", 15*time.Second),
		CleanupInterval: getEnvDuration("CLEANUP_INTERVAL", 5*time.Second),
		RequestTimeout:  getEnvDuration("REQUEST_TIMEOUT", 30*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
