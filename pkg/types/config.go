package types

import (
	"encoding/json"
	"time"
)

// LoadPattern defines how load increases or behaves over time.
type LoadPattern string

const (
	PatternConstant      LoadPattern = "constant"
	PatternRamp          LoadPattern = "ramp"
	PatternSpike         LoadPattern = "spike"
	PatternStep          LoadPattern = "step"
	PatternBreakToCrash  LoadPattern = "break-to-crash" // Ramps until server crashes or trips failure criteria
)

// StressConfig specifies options when running in BreakUntilCrash or Stress mode.
type StressConfig struct {
	Enabled               bool          `json:"enabled" yaml:"enabled"`                             // activate break-to-crash mode
	InitialConcurrency    int           `json:"initialConcurrency" yaml:"initialConcurrency"`       // e.g. 5
	StepConcurrency       int           `json:"stepConcurrency" yaml:"stepConcurrency"`             // concurrency increment per step, e.g. 10
	StepInterval          time.Duration `json:"stepInterval" yaml:"stepInterval"`                   // duration per step, e.g. 3s
	MaxConcurrency        int           `json:"maxConcurrency" yaml:"maxConcurrency"`               // upper safety limit (0 for 50,000 ceiling)
	CrashErrorRate        float64       `json:"crashErrorRate" yaml:"crashErrorRate"`               // error rate considered total server down (e.g. 75.0%)
	CrashConsecutiveFails int           `json:"crashConsecutiveFails" yaml:"crashConsecutiveFails"` // consecutive drops considered crash (e.g. 30)
	CrashMaxLatency       time.Duration `json:"crashMaxLatency" yaml:"crashMaxLatency"`             // latency considered freeze/unresponsive (e.g. 15s)
}

// AuthConfig defines authentication details for requests.
type AuthConfig struct {
	Type     string            `json:"type" yaml:"type"`         // "bearer", "basic", "apiKey", "custom"
	Token    string            `json:"token" yaml:"token"`       // for bearer
	Username string            `json:"username" yaml:"username"` // for basic
	Password string            `json:"password" yaml:"password"` // for basic
	Key      string            `json:"key" yaml:"key"`           // header or query key
	Value    string            `json:"value" yaml:"value"`       // header or query value
	In       string            `json:"in" yaml:"in"`             // "header" or "query"
	Headers  map[string]string `json:"headers" yaml:"headers"`   // custom headers
}

// EndpointSpec defines an HTTP endpoint to test.
type EndpointSpec struct {
	Name           string            `json:"name" yaml:"name"`
	URL            string            `json:"url" yaml:"url"`
	Method         string            `json:"method" yaml:"method"` // GET, POST, PUT, DELETE, etc.
	Headers        map[string]string `json:"headers" yaml:"headers"`
	Body           string            `json:"body" yaml:"body"`
	Auth           *AuthConfig       `json:"auth,omitempty" yaml:"auth,omitempty"`
	Timeout        time.Duration     `json:"timeout" yaml:"timeout"`
	ExpectedStatus []int             `json:"expectedStatus" yaml:"expectedStatus"` // e.g. [200, 201]
	Weight         int               `json:"weight" yaml:"weight"`                 // relative traffic weight (default 1)
}

// CircuitBreakerConfig defines safety rules to abort tests before breaking services.
type CircuitBreakerConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled"`
	MaxErrorRate     float64       `json:"maxErrorRate" yaml:"maxErrorRate"`         // e.g. 50.0 (abort if >= 50% errors)
	MaxLatency       time.Duration `json:"maxLatency" yaml:"maxLatency"`             // e.g. 10s (abort if avg latency > 10s)
	ConsecutiveFails int           `json:"consecutiveFails" yaml:"consecutiveFails"` // abort after N consecutive failed requests
}

// Thresholds defines performance SLAs for pass/fail decisions.
type Thresholds struct {
	MaxP95        time.Duration `json:"maxP95" yaml:"maxP95"`                   // e.g. 200ms
	MaxP99        time.Duration `json:"maxP99" yaml:"maxP99"`                   // e.g. 500ms
	MaxAvgLatency time.Duration `json:"maxAvgLatency" yaml:"maxAvgLatency"`     // e.g. 100ms
	MaxErrorRate  float64       `json:"maxErrorRate" yaml:"maxErrorRate"`       // e.g. 1.0 (1%)
	MinThroughput float64       `json:"minThroughput" yaml:"minThroughput"`     // e.g. 500 RPS
}

// MonitorConfig configures system metric scraping.
type MonitorConfig struct {
	Enabled    bool          `json:"enabled" yaml:"enabled"`
	Interval   time.Duration `json:"interval" yaml:"interval"`     // e.g. 1s
	MetricsURL string        `json:"metricsUrl" yaml:"metricsUrl"` // optional Prometheus /metrics endpoint
	LocalHost  bool          `json:"localHost" yaml:"localHost"`   // collect host CPU/RAM
}

// Config is the top-level configuration for a load test suite.
type Config struct {
	Name             string               `json:"name" yaml:"name"`
	BaseURL          string               `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"` // Base URL for relative endpoint paths
	Endpoints        []EndpointSpec       `json:"endpoints" yaml:"endpoints"`
	Duration         time.Duration        `json:"duration" yaml:"duration"`
	Concurrency      int                  `json:"concurrency" yaml:"concurrency"` // Virtual Users / Goroutine workers
	RampUp           time.Duration        `json:"rampUp" yaml:"rampUp"`
	RampDown         time.Duration        `json:"rampDown" yaml:"rampDown"`
	TargetRPS        float64              `json:"targetRps" yaml:"targetRps"` // 0 = unlimited
	Pattern          LoadPattern          `json:"pattern" yaml:"pattern"`
	Stress           StressConfig         `json:"stress" yaml:"stress"`
	CircuitBreaker   CircuitBreakerConfig `json:"circuitBreaker" yaml:"circuitBreaker"`
	Thresholds       Thresholds           `json:"thresholds" yaml:"thresholds"`
	Monitor          MonitorConfig        `json:"monitor" yaml:"monitor"`
	OutputFormat     string               `json:"outputFormat" yaml:"outputFormat"` // "json", "html", "all"
	OutputPath       string               `json:"outputPath" yaml:"outputPath"`
	// AllowInsecureTLS disables TLS certificate verification.
	// SECURITY WARNING: only enable this in controlled test environments against
	// internal/self-signed servers. Never use against production HTTPS endpoints.
	AllowInsecureTLS bool `json:"allowInsecureTls" yaml:"allowInsecureTls"`
}

// DefaultConfig provides sensible, production-ready defaults.
func DefaultConfig() Config {
	return Config{
		Name:        "API Load Test",
		Duration:    10 * time.Second,
		Concurrency: 10,
		RampUp:      2 * time.Second,
		RampDown:    1 * time.Second,
		Pattern:     PatternConstant,
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:          true,
			MaxErrorRate:     50.0,
			MaxLatency:       10 * time.Second,
			ConsecutiveFails: 100,
		},
		Thresholds: Thresholds{
			MaxP95:        300 * time.Millisecond,
			MaxP99:        1 * time.Second,
			MaxAvgLatency: 150 * time.Millisecond,
			MaxErrorRate:  2.0,
			MinThroughput: 10.0,
		},
		Monitor: MonitorConfig{
			Enabled:   true,
			Interval:  1 * time.Second,
			LocalHost: true,
		},
		OutputFormat:     "all",
		OutputPath:       "./results",
		AllowInsecureTLS: false, // secure by default
	}
}

// Clone creates a deep copy of Config.
func (c Config) Clone() Config {
	data, _ := json.Marshal(c)
	var clone Config
	_ = json.Unmarshal(data, &clone)
	return clone
}
