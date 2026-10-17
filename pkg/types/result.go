package types

import (
	"time"
)

// LatencyStats holds statistical distribution of request latencies.
type LatencyStats struct {
	Min    time.Duration `json:"min"`
	Max    time.Duration `json:"max"`
	Mean   time.Duration `json:"mean"`
	Median time.Duration `json:"median"` // p50
	P90    time.Duration `json:"p90"`
	P95    time.Duration `json:"p95"`
	P99    time.Duration `json:"p99"`
	StdDev time.Duration `json:"stdDev"`
}

// EndpointResult contains metrics for an individual endpoint.
type EndpointResult struct {
	Name            string         `json:"name"`
	URL             string         `json:"url"`
	Method          string         `json:"method"`
	TotalRequests   int64          `json:"totalRequests"`
	SuccessRequests int64          `json:"successRequests"`
	FailedRequests  int64          `json:"failedRequests"`
	ErrorRate       float64        `json:"errorRate"` // percentage 0-100
	RPS             float64        `json:"rps"`
	Latency         LatencyStats   `json:"latency"`
	StatusCodes     map[int]int64  `json:"statusCodes"`
	Errors          map[string]int `json:"errors"`
}

// TimeSeriesPoint records snapshot metrics over a time bucket (e.g. per second).
type TimeSeriesPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	RelativeSec    float64   `json:"relativeSec"`
	Requests       int64     `json:"requests"`
	Successes      int64     `json:"successes"`
	Failures       int64     `json:"failures"`
	RPS            float64   `json:"rps"`
	AvgLatencyMs   float64   `json:"avgLatencyMs"`
	P95LatencyMs   float64   `json:"p95LatencyMs"`
	ActiveUsers    int       `json:"activeUsers"`
	CPUPercent     float64   `json:"cpuPercent,omitempty"`
	MemoryMB       float64   `json:"memoryMb,omitempty"`
}

// ThresholdEvaluation records whether an SLA rule passed or failed.
type ThresholdEvaluation struct {
	Metric   string  `json:"metric"`
	Target   string  `json:"target"`
	Actual   string  `json:"actual"`
	Passed   bool    `json:"passed"`
	Severity string  `json:"severity"` // "CRITICAL", "WARNING", "INFO"
}

// BottleneckInsight represents an automated diagnostic recommendation.
type BottleneckInsight struct {
	Title       string `json:"title"`
	Severity    string `json:"severity"` // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
	Suggestion  string `json:"suggestion"`
}

// SystemMetrics tracks host or container utilization during the test.
type SystemMetrics struct {
	AverageCPUPercent float64 `json:"avgCpuPercent"`
	PeakCPUPercent    float64 `json:"peakCpuPercent"`
	AverageMemoryMB   float64 `json:"avgMemoryMb"`
	PeakMemoryMB      float64 `json:"peakMemoryMb"`
}

// TestResult is the complete test report output.
type TestResult struct {
	SuiteName       string                 `json:"suiteName"`
	StartTime       time.Time              `json:"startTime"`
	EndTime         time.Time              `json:"endTime"`
	Duration        time.Duration          `json:"duration"`
	TotalRequests   int64                  `json:"totalRequests"`
	SuccessRequests int64                  `json:"successRequests"`
	FailedRequests  int64                  `json:"failedRequests"`
	ErrorRate       float64                `json:"errorRate"`
	ThroughputRPS   float64                `json:"throughputRps"`
	Latency         LatencyStats           `json:"latency"`
	StatusCodes     map[int]int64          `json:"statusCodes"`
	ErrorSummary    map[string]int         `json:"errorSummary"`
	Endpoints       []EndpointResult       `json:"endpoints"`
	Timeline        []TimeSeriesPoint      `json:"timeline"`
	System          SystemMetrics          `json:"system"`
	Thresholds      []ThresholdEvaluation  `json:"thresholds"`
	Passed          bool                   `json:"passed"`
	Bottlenecks     []BottleneckInsight    `json:"bottlenecks"`
	Aborted         bool                   `json:"aborted"`
	AbortReason     string                 `json:"abortReason,omitempty"`
}
