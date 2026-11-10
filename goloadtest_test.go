package goloadtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func TestRoot_Run(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Name = "Root Package Integration Test"
	cfg.Duration = 1 * time.Second
	cfg.Concurrency = 5
	cfg.Endpoints = []types.EndpointSpec{
		{Name: "MockAPI", URL: server.URL, Method: "GET"},
	}
	cfg.Thresholds = types.Thresholds{
		MaxP95:       1 * time.Second,
		MaxErrorRate: 1.0,
	}
	cfg.OutputFormat = "none" // don't write files during unit test

	res, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalRequests == 0 {
		t.Fatalf("expected requests > 0")
	}
	if !res.Passed {
		t.Fatalf("expected test to pass")
	}
}
