package balancer

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/xinnaider/flux/internal/registry"
)

// ProxyConfig holds configuration for the reverse proxy.
type ProxyConfig struct {
	Timeout        time.Duration
	MaxIdleConns   int
	MaxIdlePerHost int
}

// Proxy is a reverse proxy that routes requests to the best instance
// registered in the service registry. Replaces the 302 redirect mode.
type Proxy struct {
	registry  registry.Registry
	transport *http.Transport
}

// NewProxy creates a reverse proxy that uses the registry for service discovery.
func NewProxy(reg registry.Registry, cfg ProxyConfig) *Proxy {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdlePerHost,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	return &Proxy{
		registry:  reg,
		transport: transport,
	}
}

// ServeHTTP implements http.Handler.
// It extracts the service name from the URL path, picks the best instance,
// rewrites the request URL, forwards proxy headers, and reverse-proxies
// the request to the backend.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract service name: /{name}/path
	origPath := r.URL.Path
	path := strings.TrimPrefix(origPath, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "service name is required", http.StatusBadRequest)
		return
	}
	serviceName := parts[0]

	// Pick the best instance (least connections)
	inst, err := p.registry.GetInstance(r.Context(), serviceName)
	if err != nil {
		log.Printf("[proxy] no instance for %q: %v", serviceName, err)
		http.Error(w, fmt.Sprintf("no available instances for %q", serviceName), http.StatusServiceUnavailable)
		return
	}

	// Target path: strip service name prefix
	targetPath := "/"
	if len(parts) > 1 {
		targetPath = "/" + parts[1]
	}

	// Forward proxy headers
	p.setForwardedHeaders(r)

	// Create per-request ReverseProxy with Director closure.
	// httputil.ReverseProxy requires either Director or Rewrite set (Go 1.22+).
	rp := &httputil.ReverseProxy{
		Transport: p.transport,
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = fmt.Sprintf("%s:%d", inst.Host, inst.Port)
			req.URL.Path = targetPath
			// RawQuery is preserved from the cloned request automatically
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[proxy] error proxying to %s: %v", r.URL.Host, err)
			http.Error(w, "bad gateway", http.StatusBadGateway)
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Del("Server")
			return nil
		},
	}

	rp.ServeHTTP(w, r)
}

func (p *Proxy) setForwardedHeaders(r *http.Request) {
	// X-Forwarded-For
	if clientIP := r.Header.Get("X-Forwarded-For"); clientIP != "" {
		r.Header.Set("X-Forwarded-For", clientIP+", "+r.RemoteAddr)
	} else {
		r.Header.Set("X-Forwarded-For", r.RemoteAddr)
	}

	// X-Forwarded-Host
	r.Header.Set("X-Forwarded-Host", r.Host)

	// X-Forwarded-Proto
	fp := r.Header.Get("X-Forwarded-Proto")
	if fp == "" {
		if r.TLS != nil {
			fp = "https"
		} else {
			fp = "http"
		}
	}
	r.Header.Set("X-Forwarded-Proto", fp)

	// X-Real-IP (common for nginx -> backend)
	r.Header.Set("X-Real-IP", r.RemoteAddr)
}
