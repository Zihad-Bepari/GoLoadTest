package metrics

import (
	"testing"
	"time"

	"github.com/goloadtest/goloadtest/pkg/requester"
	"github.com/goloadtest/goloadtest/pkg/types"
)

func TestCollector_BasicAggregation(t *testing.T) {
	endpoints := []types.EndpointSpec{
		{Name: "API1", URL: "http://example.com/api1", Method: "GET"},
	}
	collector := NewCollector(endpoints)
	collector.Start()

	// Simulate 100 requests: 90 success (10ms - 90ms), 10 fails (500 Internal Server Error)
	for i := 1; i <= 90; i++ {
		collector.Record(requester.ResponseRecord{
			EndpointName: "API1",
			URL:          "http://example.com/api1",
			Method:       "GET",
			StatusCode:   200,
			Duration:     time.Duration(i) * time.Millisecond,
			Timestamp:    time.Now(),
			Success:      true,
		})
	}
	for i := 1; i <= 10; i++ {
		collector.Record(requester.ResponseRecord{
			EndpointName: "API1",
			URL:          "http://example.com/api1",
			Method:       "GET",
			StatusCode:   500,
			Duration:     100 * time.Millisecond,
			Timestamp:    time.Now(),
			Success:      false,
			ErrorMessage: "500 Internal Server Error",
		})
	}

	total, success, fail, errRate, avg := collector.CurrentStats()
	if total != 100 {
		t.Fatalf("expected 100 total, got %d", total)
	}
	if success != 90 {
		t.Fatalf("expected 90 success, got %d", success)
	}
	if fail != 10 {
		t.Fatalf("expected 10 fail, got %d", fail)
	}
	if errRate != 10.0 {
		t.Fatalf("expected 10%% error rate, got %f", errRate)
	}
	if avg <= 0 {
		t.Fatalf("expected avg > 0")
	}

	result := collector.Stop()
	if result.TotalRequests != 100 {
		t.Fatalf("expected 100 result requests, got %d", result.TotalRequests)
	}
	if result.Latency.Min != 1*time.Millisecond {
		t.Fatalf("expected min 1ms, got %v", result.Latency.Min)
	}
	if result.Latency.P95 <= 0 {
		t.Fatalf("expected valid P95, got %v", result.Latency.P95)
	}
	if len(result.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint result, got %d", len(result.Endpoints))
	}
}
