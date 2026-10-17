package goloadtest

import (
	"context"

	"github.com/goloadtest/goloadtest/pkg/analyzer"
	"github.com/goloadtest/goloadtest/pkg/engine"
	"github.com/goloadtest/goloadtest/pkg/reporter"
	"github.com/goloadtest/goloadtest/pkg/types"
)

// Re-export common types for seamless developer experience.
type (
	Config               = types.Config
	EndpointSpec         = types.EndpointSpec
	AuthConfig           = types.AuthConfig
	Thresholds           = types.Thresholds
	CircuitBreakerConfig = types.CircuitBreakerConfig
	MonitorConfig        = types.MonitorConfig
	LoadPattern          = types.LoadPattern
	TestResult           = types.TestResult
	LatencyStats         = types.LatencyStats
	BottleneckInsight    = types.BottleneckInsight
	ProgressSnapshot     = engine.ProgressSnapshot
)

const (
	PatternConstant = types.PatternConstant
	PatternRamp     = types.PatternRamp
	PatternSpike    = types.PatternSpike
	PatternStep     = types.PatternStep
)

// DefaultConfig returns ready-to-use configuration defaults.
func DefaultConfig() Config {
	return types.DefaultConfig()
}

// Run executes a load test suite end-to-end, performs bottleneck analysis,
// evaluates SLAs, and optionally generates reports.
func Run(ctx context.Context, cfg Config) (TestResult, error) {
	eng := engine.New(cfg)
	res, err := eng.Run(ctx)
	if err != nil {
		return res, err
	}

	// Run Bottleneck & Threshold Analyzer
	analyzer.Analyze(cfg, &res)

	// Save reports if output format is specified
	if cfg.OutputFormat != "" && cfg.OutputFormat != "none" {
		_ = reporter.SaveResults(res, cfg.OutputFormat, cfg.OutputPath)
	}

	return res, nil
}

// CompareBaseline compares two test results to detect performance regressions.
func CompareBaseline(current, baseline TestResult) analyzer.RegressionReport {
	return analyzer.CompareBaseline(current, baseline)
}
