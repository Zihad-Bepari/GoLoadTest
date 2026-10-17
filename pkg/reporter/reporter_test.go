package reporter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goloadtest/goloadtest/pkg/types"
)

func TestReporter_Generate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goloadtest_reports")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	result := types.TestResult{
		SuiteName:       "Test Suite",
		TotalRequests:   500,
		SuccessRequests: 495,
		FailedRequests:  5,
		ErrorRate:       1.0,
		ThroughputRPS:   100.5,
		Latency: types.LatencyStats{
			Min:    2 * time.Millisecond,
			Max:    50 * time.Millisecond,
			Mean:   15 * time.Millisecond,
			Median: 12 * time.Millisecond,
			P95:    30 * time.Millisecond,
			P99:    45 * time.Millisecond,
		},
		StatusCodes: map[int]int64{200: 495, 500: 5},
		Timeline: []types.TimeSeriesPoint{
			{RelativeSec: 1, Requests: 100, RPS: 100, AvgLatencyMs: 12.0},
			{RelativeSec: 2, Requests: 105, RPS: 105, AvgLatencyMs: 14.0},
		},
		Passed: true,
	}

	jsonPath := filepath.Join(tempDir, "test.json")
	if err := GenerateJSON(result, jsonPath); err != nil {
		t.Fatalf("failed to generate JSON: %v", err)
	}

	htmlPath := filepath.Join(tempDir, "test.html")
	if err := GenerateHTML(result, htmlPath); err != nil {
		t.Fatalf("failed to generate HTML: %v", err)
	}

	// Verify file exists and has content
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read html file: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("generated HTML is empty")
	}
}
