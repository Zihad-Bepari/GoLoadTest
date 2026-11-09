package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goloadtest/goloadtest/pkg/types"
)

func TestServer_Index(t *testing.T) {
	s := NewServer("127.0.0.1:0")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "GoLoadTest") {
		t.Fatalf("expected title in body, got %s", body[:200])
	}
	if !strings.Contains(body, "START ALL API") {
		t.Fatalf("expected test button in body")
	}
}

func TestServer_StartAndStatus(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	s := NewServer("127.0.0.1:0")

	payload := strings.NewReader(`{
		"url": "` + mockServer.URL + `",
		"method": "GET",
		"duration": "1s",
		"concurrency": 2,
		"breakUntilCrash": false
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/start", payload)
	w := httptest.NewRecorder()

	s.handleStart(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// Status check
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	wStatus := httptest.NewRecorder()
	s.handleStatus(wStatus, reqStatus)

	if wStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wStatus.Code)
	}

	// Wait for test to complete
	deadline := time.Now().Add(4 * time.Second)
	var lastRes *types.TestResult
	for time.Now().Before(deadline) {
		s.mu.Lock()
		lastRes = s.lastResult
		s.mu.Unlock()
		if lastRes != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if lastRes == nil {
		t.Fatalf("expected lastResult to be populated")
	}
	if lastRes.TotalRequests == 0 {
		t.Fatalf("expected total requests > 0")
	}
}
