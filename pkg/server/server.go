package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/analyzer"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/engine"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/reporter"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

// Server coordinates the simple 1-click test runner & live dashboard.
type Server struct {
	addr        string
	mu          sync.Mutex
	isRunning   bool
	cancelTest  context.CancelFunc
	currentEng  *engine.Engine
	lastResult  *types.TestResult
	subscribers map[chan string]struct{}
}

// NewServer creates a new dashboard server.
func NewServer(addr string) *Server {
	return &Server{
		addr:        addr,
		subscribers: make(map[chan string]struct{}),
	}
}

// Start launches the HTTP web server.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/start", s.handleStart)
	mux.HandleFunc("/api/stop", s.handleStop)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/events", s.handleEvents)

	server := &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
	}

	fmt.Printf("\n⚡ GoLoadTest 1-Click Studio running at: http://%s\n", s.addr)
	return server.ListenAndServe()
}

func (s *Server) broadcast(event string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(payload))
	for ch := range s.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	msgCh := make(chan string, 32)
	s.mu.Lock()
	s.subscribers[msgCh] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.subscribers, msgCh)
		s.mu.Unlock()
	}()

	notify := r.Context().Done()
	flusher.Flush()

	for {
		select {
		case <-notify:
			return
		case msg := <-msgCh:
			_, _ = w.Write([]byte(msg))
			flusher.Flush()
		}
	}
}

type OneClickStartRequest struct {
	BaseURL     string `json:"baseUrl"` // e.g. "http://localhost:5000"
	URL         string `json:"url"`     // alias for baseUrl
	Duration    string `json:"duration,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
}

// Common REST API route candidates to discover and stress-test automatically
var discoveredRoutes = []struct {
	Name   string
	Path   string
	Method string
}{
	{"Health Check", "/api/health", "GET"},
	{"Root API", "/api", "GET"},
	{"Hajj Packages", "/api/hajj-packages", "GET"},
	{"Umrah Packages", "/api/umrah-packages", "GET"},
	{"Blogs & Articles", "/api/blogs", "GET"},
	{"Photo Gallery", "/api/gallery", "GET"},
	{"Services", "/api/services", "GET"},
	{"FAQs", "/api/faqs", "GET"},
	{"Site Config", "/api/site-config", "GET"},
	{"Ziyarat Places", "/api/ziyarats", "GET"},
	{"Hotels List", "/api/hotels", "GET"},
	{"Package Categories", "/api/package-categories", "GET"},
	{"Banners & Ads", "/api/banners", "GET"},
	{"Why Choose Us", "/api/why-choose", "GET"},
	{"Umrah Guidelines", "/api/umrah-guidelines", "GET"},
	{"Umrah FAQs", "/api/umrah-faqs", "GET"},
	{"Team Members", "/api/team-members", "GET"},
	{"Stories", "/api/stories", "GET"},
	{"Privacy Policies", "/api/privacy-policies", "GET"},
	{"Utilities", "/api/utilities", "GET"},
	{"Visas Info", "/api/visas", "GET"},
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OneClickStartRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	baseURL := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	if baseURL == "" && req.URL != "" {
		baseURL = strings.TrimRight(strings.TrimSpace(req.URL), "/")
	}
	if baseURL == "" {
		baseURL = "http://localhost:5000"
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		http.Error(w, "A test is already currently running", http.StatusConflict)
		return
	}
	s.isRunning = true
	s.lastResult = nil
	s.mu.Unlock()

	// Build endpoint list automatically
	var endpoints []types.EndpointSpec
	for _, route := range discoveredRoutes {
		endpoints = append(endpoints, types.EndpointSpec{
			Name:   route.Name,
			URL:    baseURL + route.Path,
			Method: route.Method,
			Weight: 1,
		})
	}

	// Automatic break-until-crash config
	cfg := types.DefaultConfig()
	cfg.Name = "Autonomous All-API Stress Test"
	cfg.Endpoints = endpoints
	cfg.Duration = 60 * time.Second
	if req.Duration != "" {
		if d, err := time.ParseDuration(req.Duration); err == nil && d > 0 {
			cfg.Duration = d
		}
	}
	cfg.Concurrency = 200 // Max capacity ceiling
	if req.Concurrency > 0 {
		cfg.Concurrency = req.Concurrency
	}
	cfg.Pattern = types.PatternBreakToCrash
	cfg.Stress = types.StressConfig{
		Enabled:               true,
		InitialConcurrency:    5,
		StepConcurrency:       15,
		StepInterval:          2 * time.Second,
		CrashErrorRate:        65.0,
		CrashConsecutiveFails: 20,
		CrashMaxLatency:       8 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancelTest = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.isRunning = false
			s.mu.Unlock()
		}()

		eng := engine.New(cfg)
		s.mu.Lock()
		s.currentEng = eng
		s.mu.Unlock()

		eng.SetProgressListener(func(snap engine.ProgressSnapshot) {
			s.broadcast("progress", snap)
		})

		s.broadcast("started", map[string]any{
			"baseUrl":       baseURL,
			"endpointsCount": len(endpoints),
		})

		res, err := eng.Run(ctx)
		if err == nil {
			analyzer.Analyze(cfg, &res)
			_ = reporter.SaveResults(res, "all", "./results")

			s.mu.Lock()
			s.lastResult = &res
			s.mu.Unlock()

			s.broadcast("completed", res)
		} else {
			s.broadcast("error", map[string]string{"error": err.Error()})
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":         "started",
		"baseUrl":        baseURL,
		"endpointsCount": len(endpoints),
	})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.isRunning && s.cancelTest != nil {
		s.cancelTest()
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopping"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"isRunning":  s.isRunning,
		"lastResult": s.lastResult,
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(getDashboardHTML()))
}
