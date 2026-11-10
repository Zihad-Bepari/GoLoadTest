package main

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func main() {
	var userCounter int64

	// 1. Setup mock Gin-style HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`))
		case http.MethodPost:
			atomic.AddInt64(&userCounter, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":%d,"status":"created"}`, atomic.LoadInt64(&userCounter))))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	server := &http.Server{
		Addr:    ":8085",
		Handler: mux,
	}

	go func() {
		fmt.Println("🚀 Gin/Go API Server listening on http://localhost:8085")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()
	time.Sleep(200 * time.Millisecond) // Allow server to start

	// 2. Configure GoLoadTest
	cfg := goloadtest.DefaultConfig()
	cfg.Name = "Gin Users API Benchmark"
	cfg.Duration = 3 * time.Second
	cfg.Concurrency = 20
	cfg.Pattern = types.PatternRamp
	cfg.RampUp = 500 * time.Millisecond
	cfg.OutputPath = "./results"
	cfg.OutputFormat = "all"

	cfg.Endpoints = []types.EndpointSpec{
		{
			Name:           "Get Users",
			URL:            "http://localhost:8085/api/users",
			Method:         "GET",
			Weight:         4,
			ExpectedStatus: []int{200},
		},
		{
			Name:           "Create User",
			URL:            "http://localhost:8085/api/users",
			Method:         "POST",
			Body:           `{"name":"Charlie","email":"charlie@example.com"}`,
			Headers:        map[string]string{"Content-Type": "application/json"},
			Weight:         1,
			ExpectedStatus: []int{201},
		},
	}

	cfg.Thresholds = types.Thresholds{
		MaxP95:        50 * time.Millisecond,
		MaxErrorRate:  1.0,
		MinThroughput: 100.0,
	}

	fmt.Println("\n⚡ Running GoLoadTest native Go benchmark...")
	res, err := goloadtest.Run(context.Background(), cfg)
	if err != nil {
		fmt.Printf("Benchmark error: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Finished Benchmark! Total requests: %d, Throughput: %.1f RPS, P95: %v, Passed: %v\n",
		res.TotalRequests, res.ThroughputRPS, res.Latency.P95, res.Passed)

	// Clean shutdown
	_ = server.Shutdown(context.Background())
}
