package goloadtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func BenchmarkLoadEngine(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Duration = 1 * time.Second
	cfg.Concurrency = 20
	cfg.Endpoints = []types.EndpointSpec{
		{Name: "BenchAPI", URL: server.URL, Method: "GET"},
	}
	cfg.OutputFormat = "none"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Run(context.Background(), cfg)
	}
}
