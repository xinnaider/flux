package registry

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// getRedisAddr returns the Redis address, configurable via REDIS_TEST_ADDR env var.
func getRedisAddr() string {
	if addr := os.Getenv("REDIS_TEST_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

func setupTestRedis(t *testing.T) (*RedisRegistry, func()) {
	t.Helper()

	rdb := redis.NewClient(&redis.Options{
		Addr: getRedisAddr(),
		DB:   1, // use separate DB for tests
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}

	// Clean test DB.
	rdb.FlushDB(ctx)

	reg := NewRedisRegistry(rdb, 30*time.Second)
	teardown := func() {
		rdb.FlushDB(ctx)
		rdb.Close()
	}

	return reg, teardown
}

func TestRegisterAndGetInstance(t *testing.T) {
	reg, teardown := setupTestRedis(t)
	defer teardown()

	ctx := context.Background()

	// Register two instances for the same service.
	id1, err := reg.Register(ctx, RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001, HealthURL: "/health",
	})
	if err != nil {
		t.Fatalf("register instance 1: %v", err)
	}

	id2, err := reg.Register(ctx, RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.2", Port: 3002, HealthURL: "/health",
	})
	if err != nil {
		t.Fatalf("register instance 2: %v", err)
	}

	// Send heartbeats with different connection counts.
	_ = reg.Heartbeat(ctx, HeartbeatRequest{Name: "ms.auth", InstanceID: id1, ActiveConnections: 5})
	_ = reg.Heartbeat(ctx, HeartbeatRequest{Name: "ms.auth", InstanceID: id2, ActiveConnections: 2})

	// GetInstance should return the one with fewer connections (id2).
	inst, err := reg.GetInstance(ctx, "ms.auth")
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if inst.ID != id2 {
		t.Errorf("expected instance %q (2 conns), got %q (%d conns)", id2, inst.ID, inst.ActiveConnections)
	}
}

func TestHeartbeatRenewsTTL(t *testing.T) {
	reg, teardown := setupTestRedis(t)
	defer teardown()

	ctx := context.Background()

	if _, err := reg.Register(ctx, RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001, HealthURL: "/health",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Verify instance exists.
	insts, err := reg.ListInstances(ctx, "ms.auth")
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(insts) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(insts))
	}

	// Heartbeat updates connections.
	err = reg.Heartbeat(ctx, HeartbeatRequest{
		Name: "ms.auth", InstanceID: insts[0].ID, ActiveConnections: 10,
	})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	insts, _ = reg.ListInstances(ctx, "ms.auth")
	if insts[0].ActiveConnections != 10 {
		t.Errorf("expected connections=10, got %d", insts[0].ActiveConnections)
	}
}

func TestUnregister(t *testing.T) {
	reg, teardown := setupTestRedis(t)
	defer teardown()

	ctx := context.Background()

	id, _ := reg.Register(ctx, RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	})

	err := reg.Unregister(ctx, "ms.auth", id)
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}

	_, err = reg.GetInstance(ctx, "ms.auth")
	if err == nil {
		t.Error("expected error after unregister, got nil")
	}
}

func TestRelease(t *testing.T) {
	reg, teardown := setupTestRedis(t)
	defer teardown()

	ctx := context.Background()

	id, _ := reg.Register(ctx, RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	})

	// Set connections to 5.
	if err := reg.Heartbeat(ctx, HeartbeatRequest{Name: "ms.auth", InstanceID: id, ActiveConnections: 5}); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	// Release 1.
	err := reg.Release(ctx, "ms.auth", id)
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	insts, _ := reg.ListInstances(ctx, "ms.auth")
	if insts[0].ActiveConnections != 4 {
		t.Errorf("expected connections=4 after release, got %d", insts[0].ActiveConnections)
	}
}

func TestGetInstanceNoInstances(t *testing.T) {
	reg, teardown := setupTestRedis(t)
	defer teardown()

	ctx := context.Background()

	_, err := reg.GetInstance(ctx, "ms.unknown")
	if err == nil {
		t.Error("expected error for unknown service, got nil")
	}
}

func TestListInstances(t *testing.T) {
	reg, teardown := setupTestRedis(t)
	defer teardown()

	ctx := context.Background()

	mustRegister := func(t *testing.T, req RegisterRequest) {
		t.Helper()
		if _, err := reg.Register(ctx, req); err != nil {
			t.Fatalf("register %s@%s:%d: %v", req.Name, req.Host, req.Port, err)
		}
	}
	mustRegister(t, RegisterRequest{Name: "ms.auth", Host: "10.0.0.1", Port: 3001})
	mustRegister(t, RegisterRequest{Name: "ms.auth", Host: "10.0.0.2", Port: 3002})
	mustRegister(t, RegisterRequest{Name: "ms.other", Host: "10.0.0.3", Port: 3003})

	insts, err := reg.ListInstances(ctx, "ms.auth")
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(insts) != 2 {
		t.Errorf("expected 2 instances for ms.auth, got %d", len(insts))
	}

	insts, err = reg.ListInstances(ctx, "ms.other")
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(insts) != 1 {
		t.Errorf("expected 1 instance for ms.other, got %d", len(insts))
	}
}
