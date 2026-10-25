package monitor

import (
	"context"
	"io"
	"net/http"
	"runtime"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync"
	"time"

	rmetrics "github.com/goloadtest/goloadtest/pkg/metrics"
	"github.com/goloadtest/goloadtest/pkg/types"
)

// Monitor collects system resources (CPU, RAM) during test execution.
type Monitor struct {
	cfg       types.MonitorConfig
	collector *rmetrics.Collector
	client    *http.Client
	stopCh    chan struct{}
	wg        sync.WaitGroup

	mu      sync.Mutex
	cpuSum  float64
	cpuPeak float64
	ramSum  float64
	ramPeak float64
	samples int

	// CPU delta tracking for accurate % calculation
	prevCPUSecs     float64
	prevCPUTime     time.Time
	cpuInitialized  bool
}

// New creates a new monitor.
func New(cfg types.MonitorConfig, collector *rmetrics.Collector) *Monitor {
	if cfg.Interval <= 0 {
		cfg.Interval = 1 * time.Second
	}
	return &Monitor{
		cfg:       cfg,
		collector: collector,
		client:    &http.Client{Timeout: 3 * time.Second},
		stopCh:    make(chan struct{}),
	}
}

// Start begins periodic background sampling.
func (m *Monitor) Start(ctx context.Context) {
	if !m.cfg.Enabled {
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.cfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				cpu, ram := m.sample()
				m.record(cpu, ram)
			}
		}
	}()
}

// Stop terminates sampling and computes overall summary.
func (m *Monitor) Stop() types.SystemMetrics {
	if !m.cfg.Enabled {
		return types.SystemMetrics{}
	}

	close(m.stopCh)
	m.wg.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	var avgCPU, avgRAM float64
	if m.samples > 0 {
		avgCPU = m.cpuSum / float64(m.samples)
		avgRAM = m.ramSum / float64(m.samples)
	}

	return types.SystemMetrics{
		AverageCPUPercent: avgCPU,
		PeakCPUPercent:    m.cpuPeak,
		AverageMemoryMB:   avgRAM,
		PeakMemoryMB:      m.ramPeak,
	}
}

func (m *Monitor) record(cpu float64, ram float64) {
	m.mu.Lock()
	m.samples++
	m.cpuSum += cpu
	m.ramSum += ram
	if cpu > m.cpuPeak {
		m.cpuPeak = cpu
	}
	if ram > m.ramPeak {
		m.ramPeak = ram
	}
	m.mu.Unlock()

	if m.collector != nil {
		m.collector.RecordSystemSample(cpu, ram, 0)
	}
}

func (m *Monitor) sample() (cpuPercent float64, memoryMB float64) {
	// If a Prometheus /metrics URL is provided, try that first.
	if m.cfg.MetricsURL != "" {
		cpu, ram, ok := m.scrapePrometheus(m.cfg.MetricsURL)
		if ok {
			return cpu, ram
		}
	}

	// --- Accurate CPU % via runtime/metrics (Go 1.16+) ---
	cpuPercent = m.sampleRuntimeCPU()

	// --- Memory: use runtime.MemStats (Sys = total bytes obtained from OS) ---
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)
	memoryMB = float64(mstats.Sys) / 1024.0 / 1024.0

	return cpuPercent, memoryMB
}

// sampleRuntimeCPU returns the Go process CPU utilisation as a percentage (0–100)
// by reading the "/cpu/classes/total:cpu-seconds" runtime metric and computing
// the delta between two successive samples relative to elapsed wall time and CPU count.
func (m *Monitor) sampleRuntimeCPU() float64 {
	samples := []metrics.Sample{
		{Name: "/cpu/classes/total:cpu-seconds"},
	}
	metrics.Read(samples)

	now := time.Now()

	// Metric not available on this platform — fall back to goroutine estimate.
	if samples[0].Value.Kind() != metrics.KindFloat64 {
		return m.goroutineEstimateCPU()
	}

	currentCPUSecs := samples[0].Value.Float64()

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.cpuInitialized {
		// First sample: just record baseline, return 0.
		m.prevCPUSecs = currentCPUSecs
		m.prevCPUTime = now
		m.cpuInitialized = true
		return 0
	}

	elapsed := now.Sub(m.prevCPUTime).Seconds()
	if elapsed <= 0 {
		return 0
	}

	cpuUsed := currentCPUSecs - m.prevCPUSecs
	m.prevCPUSecs = currentCPUSecs
	m.prevCPUTime = now

	numCPUs := float64(runtime.NumCPU())
	pct := (cpuUsed / elapsed) / numCPUs * 100.0
	if pct > 100.0 {
		pct = 100.0
	}
	if pct < 0 {
		pct = 0
	}
	return pct
}

// goroutineEstimateCPU provides a coarse CPU estimate when runtime/metrics
// is unavailable (unusual, but defensive). The estimate is explicitly approximate.
func (m *Monitor) goroutineEstimateCPU() float64 {
	numGoroutines := runtime.NumGoroutine()
	maxProcs := runtime.GOMAXPROCS(0)
	// Each processor can reasonably multiplex ~50 goroutines before saturation.
	approxCPU := (float64(numGoroutines) / float64(maxProcs*50)) * 100.0
	if approxCPU > 100.0 {
		approxCPU = 100.0
	}
	if approxCPU < 1.0 {
		approxCPU = 1.0
	}
	return approxCPU
}

func (m *Monitor) scrapePrometheus(url string) (float64, float64, bool) {
	resp, err := m.client.Get(url) //nolint:noctx
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return 0, 0, false
	}

	lines := strings.Split(string(body), "\n")
	var cpuVal, memVal float64
	var foundCPU, foundMem bool

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		if (strings.Contains(key, "process_cpu") || strings.Contains(key, "cpu_usage")) && !foundCPU {
			cpuVal = val * 100.0
			foundCPU = true
		} else if (strings.Contains(key, "process_resident_memory_bytes") || strings.Contains(key, "memory_bytes")) && !foundMem {
			memVal = val / 1024.0 / 1024.0
			foundMem = true
		}
	}

	return cpuVal, memVal, (foundCPU || foundMem)
}
