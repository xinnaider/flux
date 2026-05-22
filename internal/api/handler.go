package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/xinnaider/flux/internal/balancer"
	"github.com/xinnaider/flux/internal/registry"
)

type Handler struct {
	registry registry.Registry
	proxy    *balancer.Proxy
}

func NewHandler(r registry.Registry) *Handler {
	return &Handler{registry: r}
}

// SetProxy enables reverse proxy mode. When set, the catch-all handler
// will proxy requests to backends instead of returning a 302 redirect.
func (h *Handler) SetProxy(p *balancer.Proxy) {
	h.proxy = p
}

// Register handles POST /register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req registry.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Host == "" || req.Port == 0 {
		writeError(w, http.StatusBadRequest, "name, host, and port are required")
		return
	}

	instanceID, err := h.registry.Register(r.Context(), req)
	if err != nil {
		log.Printf("[api] register error: %v", err)
		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"instance_id": instanceID,
		"ttl_seconds": 15,
	})
}

// Unregister handles POST /unregister
func (h *Handler) Unregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Name       string `json:"name"`
		InstanceID string `json:"instance_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.registry.Unregister(r.Context(), req.Name, req.InstanceID); err != nil {
		log.Printf("[api] unregister error: %v", err)
		writeError(w, http.StatusInternalServerError, "unregister failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// Heartbeat handles POST /heartbeat
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req registry.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.registry.Heartbeat(r.Context(), req); err != nil {
		log.Printf("[api] heartbeat error: %v", err)
		writeError(w, http.StatusInternalServerError, "heartbeat failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":          true,
		"ttl_seconds": 15,
	})
}

// Release handles POST /release
func (h *Handler) Release(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req registry.ReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.registry.Release(r.Context(), req.Name, req.InstanceID); err != nil {
		log.Printf("[api] release error: %v", err)
		writeError(w, http.StatusInternalServerError, "release failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// Health handles GET /health — registry's own health check.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
	})
}

// Redirect handles GET /{name}/* — the core redirect endpoint.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract service name from path: /{name}/...
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "service name is required")
		return
	}
	serviceName := parts[0]

	inst, err := h.registry.GetInstance(r.Context(), serviceName)
	if err != nil {
		log.Printf("[api] redirect: no instance for %q: %v", serviceName, err)
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("no available instances for %q", serviceName))
		return
	}

	// Build redirect URL preserving the original path and query.
	targetPath := ""
	if len(parts) > 1 {
		targetPath = "/" + parts[1]
	}
	if r.URL.RawQuery != "" {
		targetPath += "?" + r.URL.RawQuery
	}

	redirectURL := fmt.Sprintf("http://%s:%d%s", inst.Host, inst.Port, targetPath)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// RegisterRoutes sets up all HTTP routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/register", h.Register)
	mux.HandleFunc("/unregister", h.Unregister)
	mux.HandleFunc("/heartbeat", h.Heartbeat)
	mux.HandleFunc("/release", h.Release)
	mux.HandleFunc("/health", h.Health)
	// Catch-all for /{name}/... — must be registered after specific routes.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(w, http.StatusOK, map[string]string{
				"service": "loadbalancer",
				"version": "1.0.0",
			})
			return
		}
		if h.proxy != nil {
			h.proxy.ServeHTTP(w, r)
		} else {
			h.Redirect(w, r)
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[api] write json error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
