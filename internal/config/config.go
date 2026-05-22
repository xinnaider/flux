package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              int
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	HeartbeatTTL      time.Duration
	CleanupInterval   time.Duration
	RequestTimeout    time.Duration
	ProxyMode         bool
	ProxyTimeout      time.Duration
	ProxyIdleConns    int
	ProxyIdlePerHost  int
}

func Load() *Config {
	return &Config{
		Port:              getEnvInt("PORT", 8080),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		HeartbeatTTL:      getEnvDuration("HEARTBEAT_TTL", 15*time.Second),
		CleanupInterval:   getEnvDuration("CLEANUP_INTERVAL", 5*time.Second),
		RequestTimeout:    getEnvDuration("REQUEST_TIMEOUT", 30*time.Second),
		ProxyMode:         getEnvBool("PROXY_MODE", false),
		ProxyTimeout:      getEnvDuration("PROXY_TIMEOUT", 30*time.Second),
		ProxyIdleConns:    getEnvInt("PROXY_IDLE_CONNS", 100),
		ProxyIdlePerHost:  getEnvInt("PROXY_IDLE_PER_HOST", 10),
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

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
