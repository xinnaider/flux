package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jfernando/loadbalancer/internal/api"
	"github.com/jfernando/loadbalancer/internal/config"
	"github.com/jfernando/loadbalancer/internal/health"
	"github.com/jfernando/loadbalancer/internal/registry"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	// Connect to Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[main] cannot connect to redis (%s): %v", cfg.RedisAddr, err)
	}
	log.Printf("[main] connected to redis at %s", cfg.RedisAddr)

	// Initialize registry.
	reg := registry.NewRedisRegistry(rdb, cfg.HeartbeatTTL)

	// Initialize health checker.
	checker := health.NewChecker(reg, cfg.CleanupInterval)
	go checker.Start(ctx)

	// Setup HTTP server.
	handler := api.NewHandler(reg)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout,
	}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("[main] shutting down...")
		checker.Stop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[main] server shutdown error: %v", err)
		}
		rdb.Close()
	}()

	log.Printf("[main] server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[main] server error: %v", err)
	}

	log.Println("[main] server stopped")
}
