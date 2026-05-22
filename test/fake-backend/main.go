package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	fluxURL      = getEnv("FLUX_URL", "http://flux:8080")
	serviceName  = getEnv("SERVICE_NAME", "test-service")
	instanceHost = getEnv("INSTANCE_HOST", "")
	instancePort = 3000
	instanceID   string
	httpClient   = &http.Client{Timeout: 5 * time.Second}
	reqCount     int64
)

func init() {
	p := getEnv("INSTANCE_PORT", "3000")
	_, _ = fmt.Sscanf(p, "%d", &instancePort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func register() error {
	host := instanceHost
	if host == "" {
		host, _ = os.Hostname()
		if host == "" {
			host = "unknown"
		}
	}

	body := map[string]interface{}{
		"name":       serviceName,
		"host":       host,
		"port":       instancePort,
		"health_url": "/health",
	}
	log.Printf("registering: host=%s port=%d", host, instancePort)
	data, _ := json.Marshal(body)

	resp, err := httpClient.Post(fluxURL+"/register", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		InstanceID string `json:"instance_id"`
		TTLSeconds int    `json:"ttl_seconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	instanceID = result.InstanceID
	log.Printf("[%s] registered id=%s ttl=%ds", serviceName, instanceID, result.TTLSeconds)
	return nil
}

func heartbeat() error {
	body := map[string]interface{}{
		"name":               serviceName,
		"instance_id":        instanceID,
		"active_connections": reqCount,
	}
	data, _ := json.Marshal(body)
	resp, err := httpClient.Post(fluxURL+"/heartbeat", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	resp.Body.Close()
	log.Printf("[%s] heartbeat ok", instanceID)
	return nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	reqCount++

	// simulate small processing variance (0-30ms)
	time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)

	w.Header().Set("X-Instance-ID", instanceID)
	w.Header().Set("X-Service-Name", serviceName)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"instance":"%s","service":"%s","path":"%s"}`, instanceID, serviceName, r.URL.Path)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	log.Printf("starting fake-backend on :%d", instancePort)

	// retry registration until flux is up
	for i := 0; i < 30; i++ {
		if err := register(); err == nil {
			break
		} else {
			log.Printf("retry %d/30: %v", i+1, err)
		}
		time.Sleep(2 * time.Second)
	}
	if instanceID == "" {
		log.Fatal("could not register after 30 attempts")
	}

	// heartbeat loop
	go func() {
		for range time.Tick(8 * time.Second) {
			_ = heartbeat()
		}
	}()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", rootHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", instancePort), nil))
}
