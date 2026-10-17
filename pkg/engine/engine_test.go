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
