package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jfernando/loadbalancer/internal/registry"
	"github.com/redis/go-redis/v9"
)

// noRedirectClient returns an HTTP client that does NOT follow redirects.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func getRedisAddr() string {
	if addr := os.Getenv("REDIS_TEST_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

func setupTestServer(t *testing.T) (*httptest.Server, *registry.RedisRegistry, func()) {
	t.Helper()

	rdb := redis.NewClient(&redis.Options{
		Addr: getRedisAddr(),
		DB:   2,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}

	rdb.FlushDB(ctx)

	reg := registry.NewRedisRegistry(rdb, 30*time.Second)
	handler := NewHandler(reg)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := httptest.NewServer(mux)

	teardown := func() {
		server.Close()
		rdb.FlushDB(ctx)
		rdb.Close()
	}

	return server, reg, teardown
}

func TestRegisterEndpoint(t *testing.T) {
	server, _, teardown := setupTestServer(t)
	defer teardown()

	body := `{"name":"ms.auth","host":"10.0.0.1","port":3001,"health_url":"/health"}`
	resp, err := http.Post(server.URL+"/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if _, ok := result["instance_id"]; !ok {
		t.Error("missing instance_id in response")
	}
}

func TestRegisterMissingFields(t *testing.T) {
	server, _, teardown := setupTestServer(t)
	defer teardown()

	body := `{"name":"ms.auth"}`
	resp, err := http.Post(server.URL+"/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHeartbeatEndpoint(t *testing.T) {
	server, reg, teardown := setupTestServer(t)
	defer teardown()

	ctx := context.Background()
	id, _ := reg.Register(ctx, registry.RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	})

	body := `{"name":"ms.auth","instance_id":"` + id + `","active_connections":42}`
	resp, err := http.Post(server.URL+"/heartbeat", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	insts, _ := reg.ListInstances(ctx, "ms.auth")
	if len(insts) > 0 && insts[0].ActiveConnections != 42 {
		t.Errorf("expected connections=42, got %d", insts[0].ActiveConnections)
	}
}

func TestRedirectEndpoint(t *testing.T) {
	server, reg, teardown := setupTestServer(t)
	defer teardown()

	ctx := context.Background()
	reg.Register(ctx, registry.RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	})

	client := noRedirectClient()
	resp, err := client.Get(server.URL + "/ms.auth/login")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 (Found), got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "10.0.0.1:3001") {
		t.Errorf("expected redirect to 10.0.0.1:3001, got %q", loc)
	}
}

func TestRedirectUnknownService(t *testing.T) {
	server, _, teardown := setupTestServer(t)
	defer teardown()

	resp, err := http.Get(server.URL + "/ms.unknown/foo")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	server, _, teardown := setupTestServer(t)
	defer teardown()

	resp, err2 := http.Get(server.URL + "/health")
	if err2 != nil {
		t.Fatalf("request failed: %v", err2)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRedirectPreservesQuery(t *testing.T) {
	server, reg, teardown := setupTestServer(t)
	defer teardown()

	ctx := context.Background()
	reg.Register(ctx, registry.RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	})

	client := noRedirectClient()
	resp, err := client.Get(server.URL + "/ms.auth/callback?code=abc&state=123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "code=abc") || !strings.Contains(loc, "state=123") {
		t.Errorf("redirect should preserve query params, got %q", loc)
	}
}
