package monitor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goloadtest/goloadtest/pkg/types"
)

// TelemetryPayload represents an expected incoming JSON from an instrumented server.
type TelemetryPayload struct {
	Server struct {
		CPUPercent       float64 `json:"cpuPercent"`
		CPU              float64 `json:"cpu"`
		MemoryMB         float64 `json:"memoryMb"`
		RAM              float64 `json:"ram"`
		ActiveGoroutines int64   `json:"goroutines"`
		Status           string  `json:"status"`
	} `json:"server"`

	LoadGuard struct {
		ActiveInFlight   int64 `json:"activeInFlight"`
		InFlight         int64 `json:"inFlight"`
		MaxConcurrency   int64 `json:"maxConcurrency"`
		QueueLength      int64 `json:"queueLength"`
		QueueSize        int64 `json:"queueSize"`
		QueueCapacity    int64 `json:"queueCapacity"`
		MaxQueueSize     int64 `json:"maxQueueSize"`
		RejectedRequests int64 `json:"rejectedRequests"`
		Rejected         int64 `json:"rejected"`
		AcceptedRequests int64 `json:"acceptedRequests"`
		Accepted         int64 `json:"accepted"`
	} `json:"loadGuard"`

	CacheShield struct {
		CoalescedRequests int64 `json:"coalescedRequests"`
		Coalesced         int64 `json:"coalesced"`
		SavedDBQueries    int64 `json:"savedDbQueries"`
		SavedQueries      int64 `json:"savedQueries"`
		CacheHits         int64 `json:"cacheHits"`
		Hits              int64 `json:"hits"`
		NegativeCacheHits int64 `json:"negativeCacheHits"`
		NegativeHits      int64 `json:"negativeHits"`
		ActiveDBBulkhead  int64 `json:"activeDbBulkhead"`
		BulkheadActive    int64 `json:"bulkheadActive"`
		MaxDBBulkhead     int64 `json:"maxDbBulkhead"`
	} `json:"cacheShield"`
}

// TelemetryHarvester probes and collects real-time telemetry from the target system.
type TelemetryHarvester struct {
	baseURL    string
	client     *http.Client
	mu         sync.RWMutex
	telemetry  types.ServerTelemetry
	hasRemote  bool
	stopCh     chan struct{}
	wg         sync.WaitGroup

	// Traffic-inferred fallback stats
	total503s  atomic.Int64
	total500s  atomic.Int64
	total200s  atomic.Int64
}

// NewTelemetryHarvester creates a new harvester for the given target server.
func NewTelemetryHarvester(baseURL string) *TelemetryHarvester {
	trimmed := strings.TrimRight(baseURL, "/")
	// Normalize localhost to 127.0.0.1
	if strings.Contains(trimmed, "://localhost") {
		trimmed = strings.Replace(trimmed, "://localhost", "://127.0.0.1", 1)
	}

	return &TelemetryHarvester{
		baseURL: trimmed,
		client: &http.Client{
			Timeout: 800 * time.Millisecond,
		},
		stopCh: make(chan struct{}),
		telemetry: types.ServerTelemetry{
			Detected:     true,
			Source:       "Traffic Vitals & Inferred Resilience",
			ServerStatus: "OPTIMAL",
		},
	}
}

// RecordResponse updates inferred telemetry from live traffic responses.
func (h *TelemetryHarvester) RecordResponse(statusCode int, hasRetryAfter bool) {
	if statusCode == 503 {
		h.total503s.Add(1)
	} else if statusCode >= 500 {
		h.total500s.Add(1)
	} else if statusCode >= 200 && statusCode < 400 {
		h.total200s.Add(1)
	}
}

// Start launches the background probing loop.
func (h *TelemetryHarvester) Start(ctx context.Context) {
	if h.baseURL == "" {
		return
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		candidatePaths := []string{
			"/api/shield-metrics",
			"/api/telemetry",
			"/api/health",
			"/metrics",
		}

		for {
			select {
			case <-h.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.probeCandidates(ctx, candidatePaths)
			}
		}
	}()
}

func (h *TelemetryHarvester) probeCandidates(ctx context.Context, paths []string) {
	var payload TelemetryPayload
	var foundPath string

	for _, p := range paths {
		targetURL := h.baseURL + p
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			continue
		}

		resp, err := h.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
			_ = resp.Body.Close()
			if readErr == nil {
				if err := json.Unmarshal(body, &payload); err == nil {
					foundPath = p
					break
				}
			}
		} else {
			_ = resp.Body.Close()
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if foundPath != "" {
		h.hasRemote = true
		h.telemetry.Detected = true
		h.telemetry.Source = "Direct Telemetry (" + foundPath + ")"

		// Server info
		cpu := payload.Server.CPUPercent
		if cpu == 0 {
			cpu = payload.Server.CPU
		}
		ram := payload.Server.MemoryMB
		if ram == 0 {
			ram = payload.Server.RAM
		}
		h.telemetry.CPUPercent = cpu
		h.telemetry.MemoryMB = ram
		h.telemetry.ActiveGoroutines = payload.Server.ActiveGoroutines
		if payload.Server.Status != "" {
			h.telemetry.ServerStatus = strings.ToUpper(payload.Server.Status)
		} else {
			h.telemetry.ServerStatus = "OPTIMAL"
		}

		// GoLoadGuard
		lg := payload.LoadGuard
		if lg.MaxConcurrency > 0 || lg.ActiveInFlight > 0 || lg.InFlight > 0 || lg.RejectedRequests > 0 || lg.Rejected > 0 {
			h.telemetry.GuardDetected = true
			if lg.ActiveInFlight > 0 {
				h.telemetry.InFlightRequests = lg.ActiveInFlight
			} else {
				h.telemetry.InFlightRequests = lg.InFlight
			}
			h.telemetry.MaxConcurrency = lg.MaxConcurrency
			if lg.QueueLength > 0 {
				h.telemetry.QueueLength = lg.QueueLength
			} else {
				h.telemetry.QueueLength = lg.QueueSize
			}
			if lg.QueueCapacity > 0 {
				h.telemetry.QueueCapacity = lg.QueueCapacity
			} else {
				h.telemetry.QueueCapacity = lg.MaxQueueSize
			}
			if lg.RejectedRequests > 0 {
				h.telemetry.RejectedRequests = lg.RejectedRequests
			} else {
				h.telemetry.RejectedRequests = lg.Rejected
			}
			if lg.AcceptedRequests > 0 {
				h.telemetry.AcceptedRequests = lg.AcceptedRequests
			} else {
				h.telemetry.AcceptedRequests = lg.Accepted
			}

			if h.telemetry.RejectedRequests > 0 || h.telemetry.QueueLength > 0 {
				h.telemetry.ServerStatus = "GUARDED"
			}
		}

		// GoCacheShield
		cs := payload.CacheShield
		if cs.CoalescedRequests > 0 || cs.Coalesced > 0 || cs.SavedDBQueries > 0 || cs.SavedQueries > 0 || cs.CacheHits > 0 || cs.Hits > 0 {
			h.telemetry.ShieldDetected = true
			if cs.CoalescedRequests > 0 {
				h.telemetry.CoalescedRequests = cs.CoalescedRequests
			} else {
				h.telemetry.CoalescedRequests = cs.Coalesced
			}
			if cs.SavedDBQueries > 0 {
				h.telemetry.SavedDBQueries = cs.SavedDBQueries
			} else {
				h.telemetry.SavedDBQueries = cs.SavedQueries
			}
			if cs.CacheHits > 0 {
				h.telemetry.CacheHits = cs.CacheHits
			} else {
				h.telemetry.CacheHits = cs.Hits
			}
			if cs.NegativeCacheHits > 0 {
				h.telemetry.NegativeCacheHits = cs.NegativeCacheHits
			} else {
				h.telemetry.NegativeCacheHits = cs.NegativeHits
			}
			if cs.ActiveDBBulkhead > 0 {
				h.telemetry.ActiveDBBulkhead = cs.ActiveDBBulkhead
			} else {
				h.telemetry.ActiveDBBulkhead = cs.BulkheadActive
			}
			h.telemetry.MaxDBBulkhead = cs.MaxDBBulkhead
		}
	} else if !h.hasRemote {
		// Infer from traffic
		sheds := h.total503s.Load()
		errs := h.total500s.Load()
		goods := h.total200s.Load()

		h.telemetry.Detected = true
		h.telemetry.Source = "Inferred from Traffic Signatures"

		if sheds > 0 {
			h.telemetry.GuardDetected = true
			h.telemetry.RejectedRequests = sheds
			h.telemetry.ServerStatus = "THROTTLED"
		} else if errs > 50 && errs > goods {
			h.telemetry.ServerStatus = "OVERLOADED"
		} else {
			h.telemetry.ServerStatus = "OPTIMAL"
		}
	}
}

// Snapshot returns the current snapshot of telemetry.
func (h *TelemetryHarvester) Snapshot() types.ServerTelemetry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.telemetry
}

// Stop terminates the harvester.
func (h *TelemetryHarvester) Stop() types.ServerTelemetry {
	close(h.stopCh)
	h.wg.Wait()
	return h.Snapshot()
}
