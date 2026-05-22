package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Instance represents a registered service instance.
type Instance struct {
	ID                string
	Name              string
	Host              string
	Port              int
	HealthURL         string
	ActiveConnections int64
}

// RegisterRequest is the payload for registering a new instance.
type RegisterRequest struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	HealthURL string `json:"health_url"`
}

// HeartbeatRequest is the payload for a heartbeat.
type HeartbeatRequest struct {
	Name              string `json:"name"`
	InstanceID        string `json:"instance_id"`
	ActiveConnections int64  `json:"active_connections"`
}

// ReleaseRequest is the payload for releasing a connection.
type ReleaseRequest struct {
	Name       string `json:"name"`
	InstanceID string `json:"instance_id"`
}

// Registry defines the interface for service registry operations.
type Registry interface {
	Register(ctx context.Context, req RegisterRequest) (string, error)
	Unregister(ctx context.Context, name, instanceID string) error
	Heartbeat(ctx context.Context, req HeartbeatRequest) error
	Release(ctx context.Context, name, instanceID string) error
	GetInstance(ctx context.Context, name string) (*Instance, error)
	ListInstances(ctx context.Context, name string) ([]*Instance, error)
	Cleanup(ctx context.Context) error
}

// RedisRegistry implements Registry using Redis.
type RedisRegistry struct {
	rdb        *redis.Client
	ttl        time.Duration
	instanceNS string
	serviceNS  string
}

// NewRedisRegistry creates a new Redis-backed registry.
func NewRedisRegistry(rdb *redis.Client, ttl time.Duration) *RedisRegistry {
	return &RedisRegistry{
		rdb:        rdb,
		ttl:        ttl,
		instanceNS: "instance",
		serviceNS:  "service",
	}
}

func (r *RedisRegistry) instanceKey(name, id string) string {
	return fmt.Sprintf("%s:%s:%s", r.instanceNS, name, id)
}

func (r *RedisRegistry) serviceSetKey(name string) string {
	return fmt.Sprintf("%s:%s:instances", r.serviceNS, name)
}

func (r *RedisRegistry) serviceKeyPattern() string {
	return fmt.Sprintf("%s:*:instances", r.serviceNS)
}

// Register creates a new instance entry in Redis.
func (r *RedisRegistry) Register(ctx context.Context, req RegisterRequest) (string, error) {
	instanceID := fmt.Sprintf("%s:%d", req.Host, req.Port)

	key := r.instanceKey(req.Name, instanceID)
	pipe := r.rdb.Pipeline()

	pipe.HSet(ctx, key,
		"host", req.Host,
		"port", req.Port,
		"health_url", req.HealthURL,
		"connections", 0,
	)
	pipe.Expire(ctx, key, r.ttl)
	pipe.SAdd(ctx, r.serviceSetKey(req.Name), instanceID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("register: %w", err)
	}

	return instanceID, nil
}

// Unregister removes an instance from the service set and deletes its data.
func (r *RedisRegistry) Unregister(ctx context.Context, name, instanceID string) error {
	pipe := r.rdb.Pipeline()
	pipe.SRem(ctx, r.serviceSetKey(name), instanceID)
	pipe.Del(ctx, r.instanceKey(name, instanceID))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("unregister: %w", err)
	}
	return nil
}

// Heartbeat renews the TTL and updates the active connections count.
func (r *RedisRegistry) Heartbeat(ctx context.Context, req HeartbeatRequest) error {
	key := r.instanceKey(req.Name, req.InstanceID)
	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, key, "connections", req.ActiveConnections)
	pipe.Expire(ctx, key, r.ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	return nil
}

// Release decrements the active connections counter for an instance.
func (r *RedisRegistry) Release(ctx context.Context, name, instanceID string) error {
	key := r.instanceKey(name, instanceID)
	err := r.rdb.HIncrBy(ctx, key, "connections", -1).Err()
	if err != nil {
		return fmt.Errorf("release: %w", err)
	}
	return nil
}

// GetInstance picks the instance with the fewest active connections.
func (r *RedisRegistry) GetInstance(ctx context.Context, name string) (*Instance, error) {
	members, err := r.rdb.SMembers(ctx, r.serviceSetKey(name)).Result()
	if err != nil {
		return nil, fmt.Errorf("get instance (smembers): %w", err)
	}
	if len(members) == 0 {
		return nil, fmt.Errorf("no instances available for service %q", name)
	}

	var best *Instance
	var bestConns int64 = -1

	for _, id := range members {
		key := r.instanceKey(name, id)
		vals, err := r.rdb.HGetAll(ctx, key).Result()
		if err != nil || len(vals) == 0 {
			continue
		}

		port := 0
		fmt.Sscanf(vals["port"], "%d", &port)
		conns, _ := strToInt64(vals["connections"])

		if best == nil || conns < bestConns {
			best = &Instance{
				ID:                id,
				Name:              name,
				Host:              vals["host"],
				Port:              port,
				HealthURL:         vals["health_url"],
				ActiveConnections: conns,
			}
			bestConns = conns
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no healthy instances for service %q", name)
	}

	// Increment connections counter as a fallback between heartbeats.
	r.rdb.HIncrBy(ctx, r.instanceKey(name, best.ID), "connections", 1)

	return best, nil
}

// ListInstances returns all instances for a given service name.
func (r *RedisRegistry) ListInstances(ctx context.Context, name string) ([]*Instance, error) {
	members, err := r.rdb.SMembers(ctx, r.serviceSetKey(name)).Result()
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}

	var instances []*Instance
	for _, id := range members {
		key := r.instanceKey(name, id)
		vals, err := r.rdb.HGetAll(ctx, key).Result()
		if err != nil || len(vals) == 0 {
			continue
		}
		port := 0
		fmt.Sscanf(vals["port"], "%d", &port)
		conns, _ := strToInt64(vals["connections"])
		instances = append(instances, &Instance{
			ID:                id,
			Name:              name,
			Host:              vals["host"],
			Port:              port,
			HealthURL:         vals["health_url"],
			ActiveConnections: conns,
		})
	}
	return instances, nil
}

// Cleanup removes expired instances from all service sets.
// Iterates through all service:name:instances keys, checks if each
// instance's data still exists (TTL not expired), and removes stale entries.
func (r *RedisRegistry) Cleanup(ctx context.Context) error {
	var cursor uint64
	for {
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, r.serviceKeyPattern(), 100).Result()
		if err != nil {
			return fmt.Errorf("cleanup scan: %w", err)
		}
		for _, setKey := range keys {
			members, err := r.rdb.SMembers(ctx, setKey).Result()
			if err != nil {
				continue
			}
			for _, member := range members {
				// Derive name from setKey: service:{name}:instances
				var name string
				if n, err := extractName(setKey, r.serviceNS); err == nil {
					name = n
				} else {
					continue
				}
				exists, err := r.rdb.Exists(ctx, r.instanceKey(name, member)).Result()
				if err != nil {
					continue
				}
				if exists == 0 {
					r.rdb.SRem(ctx, setKey, member)
				}
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// strToInt64 parses a string to int64, returning 0 on error.
func strToInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// extractName parses the service name from a key like "service:name:instances".
func extractName(key, ns string) (string, error) {
	prefix := ns + ":"
	suffix := ":instances"
	if len(key) <= len(prefix)+len(suffix) {
		return "", fmt.Errorf("invalid key format: %s", key)
	}
	return key[len(prefix) : len(key)-len(suffix)], nil
}
