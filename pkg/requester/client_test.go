package requester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func TestClient_Execute_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "hello" {
			http.Error(w, "missing header", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(5*time.Second, 10, true)
	spec := types.EndpointSpec{
		Name:    "TestEndpoint",
		URL:     server.URL,
		Method:  "GET",
		Headers: map[string]string{"X-Custom": "hello"},
		Auth: &types.AuthConfig{
			Type:  "bearer",
			Token: "test-token",
		},
		ExpectedStatus: []int{200},
	}

	rec := client.Execute(context.Background(), spec)
	if !rec.Success {
		t.Fatalf("expected success, got error: %s", rec.ErrorMessage)
	}
	if rec.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", rec.StatusCode)
	}
	if rec.Duration <= 0 {
		t.Fatalf("expected duration > 0, got %v", rec.Duration)
	}
}

func TestClient_Execute_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(1*time.Second, 10, true)
	spec := types.EndpointSpec{
		Name:    "SlowEndpoint",
		URL:     server.URL,
		Method:  "GET",
		Timeout: 10 * time.Millisecond,
	}

	rec := client.Execute(context.Background(), spec)
	if rec.Success {
		t.Fatalf("expected timeout failure, got success")
	}
	if rec.ErrorMessage != "request_timeout" {
		t.Fatalf("expected request_timeout, got %s", rec.ErrorMessage)
	}
}
