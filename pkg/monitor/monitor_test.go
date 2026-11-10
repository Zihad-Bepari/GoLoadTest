package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/metrics"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func TestMonitor_StartStop(t *testing.T) {
	collector := metrics.NewCollector([]types.EndpointSpec{})
	collector.Start()

	cfg := types.MonitorConfig{
		Enabled:   true,
		Interval:  50 * time.Millisecond,
		LocalHost: true,
	}

	m := New(cfg, collector)
	m.Start(context.Background())

	time.Sleep(150 * time.Millisecond)

	sys := m.Stop()
	if sys.AverageMemoryMB <= 0 {
		t.Fatalf("expected memory sample > 0, got %f", sys.AverageMemoryMB)
	}
}
