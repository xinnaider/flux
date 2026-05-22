package health

import (
	"context"
	"log"
	"time"

	"github.com/xinnaider/flux/internal/registry"
)

// Checker runs periodic cleanup of expired instances.
type Checker struct {
	registry registry.Registry
	interval time.Duration
	stopCh   chan struct{}
}

// NewChecker creates a new health checker.
func NewChecker(r registry.Registry, interval time.Duration) *Checker {
	return &Checker{
		registry: r,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic cleanup loop.
func (c *Checker) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	log.Printf("[health] cleanup started (interval=%s)", c.interval)
	for {
		select {
		case <-ticker.C:
			if err := c.registry.Cleanup(ctx); err != nil {
				log.Printf("[health] cleanup error: %v", err)
			}
		case <-c.stopCh:
			log.Printf("[health] cleanup stopped")
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop signals the cleanup loop to stop.
func (c *Checker) Stop() {
	close(c.stopCh)
}
