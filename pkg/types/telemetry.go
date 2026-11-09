package types

// ServerTelemetry holds real-time system vitals and protection metrics from the target server.
type ServerTelemetry struct {
	Detected         bool    `json:"detected"`
	Source           string  `json:"source,omitempty"`         // "Direct Telemetry (/api/shield-metrics)" or "Inferred from Traffic Signatures"
	ServerStatus     string  `json:"serverStatus"`             // "OPTIMAL", "GUARDED", "THROTTLED", "OVERLOADED", "UNRESPONSIVE"
	CPUPercent       float64 `json:"cpuPercent"`
	MemoryMB         float64 `json:"memoryMb"`
	ActiveGoroutines int64   `json:"activeGoroutines,omitempty"`

	// GoLoadGuard Concurrency & Backpressure Defense
	GuardDetected    bool  `json:"guardDetected"`
	InFlightRequests int64 `json:"inFlightRequests"`
	MaxConcurrency   int64 `json:"maxConcurrency"`
	QueueLength      int64 `json:"queueLength"`
	QueueCapacity    int64 `json:"queueCapacity"`
	RejectedRequests int64 `json:"rejectedRequests"` // 503 Overload drops preventing OOM
	AcceptedRequests int64 `json:"acceptedRequests"`

	// GoCacheShield Data & Cache Defense
	ShieldDetected    bool  `json:"shieldDetected"`
	CoalescedRequests int64 `json:"coalescedRequests"` // Cache stampede collapsed
	SavedDBQueries    int64 `json:"savedDbQueries"`    // DB queries spared
	CacheHits         int64 `json:"cacheHits"`
	NegativeCacheHits int64 `json:"negativeCacheHits"` // Non-existent key attacks blocked
	ActiveDBBulkhead  int64 `json:"activeDbBulkhead"`
	MaxDBBulkhead     int64 `json:"maxDbBulkhead"`
}
