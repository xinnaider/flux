package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type result struct {
	duration   time.Duration
	statusCode int
	backendID  string
	err        error
}

type report struct {
	total      int
	failures   int
	min, max   time.Duration
	avg        time.Duration
	p50, p95   time.Duration
	p99        time.Duration
	statusCode map[int]int
	backends   map[string]int
}

var (
	testerID    = getEnv("TESTER_ID", "1")
	fluxURL     = getEnv("FLUX_URL", "http://flux:8080")
	serviceName = getEnv("SERVICE_NAME", "test-service")
	numReqs     = atoi(getEnv("NUM_REQUESTS", "500"))
	concurrency = atoi(getEnv("CONCURRENCY", "50"))
)

func getEnv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func atoi(s string) int {
	n := 0
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

func waitForFlux() {
	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; i < 60; i++ {
		resp, err := client.Get(fluxURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			log.Printf("flux ready")
			return
		}
		if err == nil {
			resp.Body.Close()
		}
		log.Printf("waiting for flux (%d/60)...", i+1)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("flux not ready after 60 attempts")
}

func waitForService() {
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < 60; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/%s/test", fluxURL, serviceName))
		if err == nil {
			resp.Body.Close()
			// proxy mode returns 200, redirect mode returns 302
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
				log.Printf("service %q has registered instances (status=%d)", serviceName, resp.StatusCode)
				return
			}
		}
		log.Printf("waiting for service %q (%d/60)...", serviceName, i+1)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("service %q not available after 60 attempts", serviceName)
}

func main() {
	log.Printf("=== LOAD TEST [tester-%s] ===", testerID)
	log.Printf("requests=%d concurrency=%d", numReqs, concurrency)
	log.Printf("flux=%s service=%s", fluxURL, serviceName)

	waitForFlux()
	waitForService()

	jobs := make(chan int, numReqs)
	for i := 0; i < numReqs; i++ {
		jobs <- i
	}
	close(jobs)

	results := make(chan result, numReqs)
	var wg sync.WaitGroup

	tr := &http.Transport{
		MaxIdleConns:        1000,
		MaxConnsPerHost:     1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     30 * time.Second,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				t0 := time.Now()
				resp, err := client.Get(fmt.Sprintf("%s/%s/test", fluxURL, serviceName))
				if err != nil {
					results <- result{err: fmt.Errorf("req: %w", err)}
					continue
				}
				backend := resp.Header.Get("X-Instance-ID")
				if backend == "" {
					backend = "unknown"
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				results <- result{
					duration:   time.Since(t0),
					statusCode: resp.StatusCode,
					backendID:  backend,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var durations []time.Duration
	statusMap := make(map[int]int)
	backendMap := make(map[string]int)
	var failCount int32

	for r := range results {
		if r.err != nil {
			atomic.AddInt32(&failCount, 1)
			fmt.Fprintf(os.Stderr, "ERR: %v\n", r.err)
			continue
		}
		durations = append(durations, r.duration)
		statusMap[r.statusCode]++
		backendMap[r.backendID]++
	}

	elapsed := time.Since(start)
	n := len(durations)

	if n == 0 {
		log.Fatal("zero successful requests")
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}

	r := report{
		total:      n,
		failures:   int(failCount),
		min:        durations[0],
		max:        durations[n-1],
		avg:        sum / time.Duration(n),
		p50:        durations[int(float64(n)*0.50)],
		p95:        durations[int(float64(n)*0.95)],
		p99:        durations[int(float64(n)*0.99)],
		statusCode: statusMap,
		backends:   backendMap,
	}

	printReport(r, elapsed)

	if float64(failCount)/float64(numReqs) > 0.5 {
		os.Exit(1)
	}
}

func printReport(r report, elapsed time.Duration) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║           LOAD TEST REPORT                   ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Duration:         %v\n", elapsed)
	fmt.Printf("  Requests:         %d total, %d successful, %d failed\n", r.total+r.failures, r.total, r.failures)
	fmt.Printf("  Throughput:       %.1f req/sec\n", float64(r.total)/elapsed.Seconds())
	fmt.Println()
	fmt.Println("  ── Latency ──")
	fmt.Printf("    Min:    %v\n", r.min)
	fmt.Printf("    Avg:    %v\n", r.avg)
	fmt.Printf("    Max:    %v\n", r.max)
	fmt.Printf("    P50:    %v\n", r.p50)
	fmt.Printf("    P95:    %v\n", r.p95)
	fmt.Printf("    P99:    %v\n", r.p99)
	fmt.Println()
	fmt.Println("  ── Status Codes ──")
	for code, count := range r.statusCode {
		pct := float64(count) / float64(r.total) * 100
		fmt.Printf("    HTTP %d: %d (%.1f%%)\n", code, count, pct)
	}
	fmt.Println()
	fmt.Println("  ── Backend Distribution ──")
	for b, count := range r.backends {
		pct := float64(count) / float64(r.total) * 100
		fmt.Printf("    %s: %d (%.1f%%)\n", b, count, pct)
	}
	fmt.Println()
}
