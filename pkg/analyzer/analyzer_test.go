package analyzer

import (
	"testing"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func TestAnalyzer_Thresholds(t *testing.T) {
	cfg := types.Config{
		Thresholds: types.Thresholds{
			MaxP95:       100 * time.Millisecond,
			MaxErrorRate: 2.0,
		},
	}

	// 1. Passing result
	resPass := &types.TestResult{
		Latency: types.LatencyStats{
			P95: 50 * time.Millisecond,
		},
		ErrorRate: 0.5,
	}
	Analyze(cfg, resPass)
	if !resPass.Passed {
		t.Fatalf("expected resPass to pass")
	}

	// 2. Failing result
	resFail := &types.TestResult{
		Latency: types.LatencyStats{
			P95: 150 * time.Millisecond, // exceeded
		},
		ErrorRate: 5.0, // exceeded
	}
	Analyze(cfg, resFail)
	if resFail.Passed {
		t.Fatalf("expected resFail to fail")
	}
	if len(resFail.Bottlenecks) == 0 {
		t.Fatalf("expected bottleneck insights to be generated")
	}
}

func TestAnalyzer_BaselineCompare(t *testing.T) {
	baseline := types.TestResult{
		ThroughputRPS: 1000,
		Latency: types.LatencyStats{
			P95: 50 * time.Millisecond,
		},
		ErrorRate: 0.1,
	}

	// Regressed run: throughput dropped to 600, p95 rose to 100ms
	current := types.TestResult{
		ThroughputRPS: 600,
		Latency: types.LatencyStats{
			P95: 100 * time.Millisecond,
		},
		ErrorRate: 3.5,
	}

	diff := CompareBaseline(current, baseline)
	if !diff.Regressed {
		t.Fatalf("expected regression detection")
	}
}
