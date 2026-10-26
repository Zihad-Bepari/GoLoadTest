package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goloadtest/goloadtest/pkg/types"
)

func TestEngine_Run_Success(t *testing.T) {
	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := types.Config{
		Name: "Test Run",
		Endpoints: []types.EndpointSpec{
			{Name: "MockAPI", URL: server.URL, Method: "GET"},
		},
		Duration:    1 * time.Second,
		Concurrency: 5,
		Pattern:     types.PatternConstant,
	}

	eng := New(cfg)
	res, err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalRequests == 0 {
		t.Fatalf("expected requests > 0, got 0")
	}
	if res.SuccessRequests != res.TotalRequests {
		t.Fatalf("expected all requests to succeed, got %d / %d. Errors: %+v", res.SuccessRequests, res.TotalRequests, res.ErrorSummary)
	}
	if res.ErrorRate != 0 {
		t.Fatalf("expected 0 error rate, got %f", res.ErrorRate)
	}
}

func TestEngine_CircuitBreaker(t *testing.T) {
	// Server always returns 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal crash", http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := types.Config{
		Name: "Circuit Breaker Test",
		Endpoints: []types.EndpointSpec{
			{Name: "CrashingAPI", URL: server.URL, Method: "GET"},
		},
		Duration:    5 * time.Second,
		Concurrency: 4,
		CircuitBreaker: types.CircuitBreakerConfig{
			Enabled:          true,
			MaxErrorRate:     50.0,
			ConsecutiveFails: 25,
		},
	}

	eng := New(cfg)
	res, err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Aborted {
		t.Fatalf("expected circuit breaker to abort test, but aborted was false")
	}
	if res.AbortReason == "" {
		t.Fatalf("expected abort reason, got empty")
	}
}

func TestEngine_BreakToCrash(t *testing.T) {
	var reqCount int64
	// Mock server that works fine for first 50 requests, then crashes (returns 503)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt64(&reqCount, 1)
		if current > 40 {
			http.Error(w, "server overloaded", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := types.Config{
		Name: "Break to Crash Test",
		Endpoints: []types.EndpointSpec{
			{Name: "DegradingAPI", URL: server.URL, Method: "GET"},
		},
		Duration:    10 * time.Second,
		Concurrency: 10,
		Pattern:     types.PatternBreakToCrash,
		Stress: types.StressConfig{
			Enabled:               true,
			InitialConcurrency:    2,
			StepConcurrency:       2,
			StepInterval:          200 * time.Millisecond,
			CrashConsecutiveFails: 10,
			CrashErrorRate:        60.0,
		},
	}

	eng := New(cfg)
	res, err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Aborted {
		t.Fatalf("expected break to crash mode to abort when server degraded")
	}
	if res.BreakingPoint == nil || !res.BreakingPoint.Detected {
		t.Fatalf("expected breaking point to be captured")
	}
	if res.BreakingPoint.CrashConcurrency <= 0 {
		t.Fatalf("expected positive crash concurrency, got %d", res.BreakingPoint.CrashConcurrency)
	}
}

func TestEngine_BaseURL_Resolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auto" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"auto_ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := types.Config{
		Name:    "BaseURL Resolution Test",
		BaseURL: server.URL,
		Endpoints: []types.EndpointSpec{
			{Name: "RelativeRoute", URL: "/api/v1/auto", Method: "GET"},
		},
		Duration:    500 * time.Millisecond,
		Concurrency: 2,
	}

	eng := New(cfg)
	res, err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalRequests == 0 || res.SuccessRequests != res.TotalRequests {
		t.Fatalf("expected all requests to succeed via BaseURL resolution, got %d / %d", res.SuccessRequests, res.TotalRequests)
	}
}
