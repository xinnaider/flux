package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xinnaider/flux/internal/balancer"
	"github.com/xinnaider/flux/internal/registry"
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
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

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
	id, err := reg.Register(ctx, registry.RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	body := `{"name":"ms.auth","instance_id":"` + id + `","active_connections":42}`
	resp, err := http.Post(server.URL+"/heartbeat", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	insts, err := reg.ListInstances(ctx, "ms.auth")
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(insts) > 0 && insts[0].ActiveConnections != 42 {
		t.Errorf("expected connections=42, got %d", insts[0].ActiveConnections)
	}
}

func TestRedirectEndpoint(t *testing.T) {
	server, reg, teardown := setupTestServer(t)
	defer teardown()

	ctx := context.Background()
	if _, err := reg.Register(ctx, registry.RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

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

func setupTestServerWithProxy(t *testing.T) (*httptest.Server, *registry.RedisRegistry, func()) {
	t.Helper()

	rdb := redis.NewClient(&redis.Options{
		Addr: getRedisAddr(),
		DB:   3,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}

	rdb.FlushDB(ctx)

	reg := registry.NewRedisRegistry(rdb, 30*time.Second)
	handler := NewHandler(reg)

	proxy := balancer.NewProxy(reg, balancer.ProxyConfig{
		MaxIdleConns:   10,
		MaxIdlePerHost: 2,
	})
	handler.SetProxy(proxy)

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

func TestProxyToBackend(t *testing.T) {
	// Start a fake backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "backend-ok")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"path":"%s","method":"%s"}`, r.URL.Path, r.Method)
	}))
	defer backend.Close()

	server, reg, teardown := setupTestServerWithProxy(t)
	defer teardown()

	ctx := context.Background()
	// Backend host:port — extract from backend.URL
	// httptest gives http://127.0.0.1:PORT
	host := strings.TrimPrefix(backend.URL, "http://")
	hostParts := strings.Split(host, ":")
	port := 0
	fmt.Sscanf(hostParts[1], "%d", &port)

	_, err := reg.Register(ctx, registry.RegisterRequest{
		Name: "ms.test", Host: hostParts[0], Port: port,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	resp, err := http.Get(server.URL + "/ms.test/hello?x=1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"path":"/hello"`) {
		t.Errorf("expected path /hello in response, got %s", body)
	}
	if resp.Header.Get("X-Test") != "backend-ok" {
		t.Errorf("expected X-Test header from backend")
	}
}

func TestProxyUnknownService(t *testing.T) {
	server, _, teardown := setupTestServerWithProxy(t)
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

func TestRedirectPreservesQuery(t *testing.T) {
	server, reg, teardown := setupTestServer(t)
	defer teardown()

	ctx := context.Background()
	if _, err := reg.Register(ctx, registry.RegisterRequest{
		Name: "ms.auth", Host: "10.0.0.1", Port: 3001,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

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
