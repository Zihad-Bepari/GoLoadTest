package metrics

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goloadtest/goloadtest/pkg/requester"
	"github.com/goloadtest/goloadtest/pkg/types"
)

// EndpointCollector aggregates metrics for a single endpoint.
type EndpointCollector struct {
	spec        types.EndpointSpec
	totalReqs   int64 // atomic
	successReqs int64 // atomic
	failedReqs  int64 // atomic

	// Protected by the parent Collector's mu
	latencies   []time.Duration
	statusCodes map[int]int64
	errors      map[string]int
}

// Collector collects and computes all metrics during a load test.
//
// Scalability design:
//   - Hot-path counters (total, success, fail, latency sum) are atomic — no lock needed for CurrentStats().
//   - The main mutex is only acquired for slice appends and map updates (timeline, status codes, errors).
//   - This dramatically reduces contention at high concurrency (1000+ virtual users).
type Collector struct {
	// Atomic hot-path counters — updated on every request, zero-contention reads.
	atomicTotal      int64
	atomicSuccess    int64
	atomicFail       int64
	atomicLatencySum int64 // nanoseconds; used for real-time avg latency

	// Protected by mu — updated less frequently, or only at Stop().
	mu              sync.Mutex
	startTime       time.Time
	endTime         time.Time
	endpoints       map[string]*EndpointCollector
	allLatencies    []time.Duration
	statusCodes     map[int]int64
	errors          map[string]int
	timeline        map[int64]*types.TimeSeriesPoint // bucketed by relative second
	bucketLatencies map[int64][]time.Duration        // per-second latency samples for chart computation
	lastSec         int64
}

// NewCollector initializes a metrics collector.
func NewCollector(endpoints []types.EndpointSpec) *Collector {
	c := &Collector{
		endpoints:       make(map[string]*EndpointCollector),
		statusCodes:     make(map[int]int64),
		errors:          make(map[string]int),
		timeline:        make(map[int64]*types.TimeSeriesPoint),
		bucketLatencies: make(map[int64][]time.Duration),
	}

	for _, ep := range endpoints {
		c.endpoints[ep.Name] = &EndpointCollector{
			spec:        ep,
			statusCodes: make(map[int]int64),
			errors:      make(map[string]int),
		}
	}

	return c
}

// Start marks the beginning of the test.
func (c *Collector) Start() {
	c.mu.Lock()
	c.startTime = time.Now()
	c.mu.Unlock()
}

// Record ingests a single request result safely from concurrent workers.
// Hot-path atomics are updated first (no lock), then the mutex is acquired
// only for slice and map mutations.
func (c *Collector) Record(rec requester.ResponseRecord) {
	// --- Lock-free hot path ---
	atomic.AddInt64(&c.atomicTotal, 1)
	atomic.AddInt64(&c.atomicLatencySum, int64(rec.Duration))
	if rec.Success {
		atomic.AddInt64(&c.atomicSuccess, 1)
	} else {
		atomic.AddInt64(&c.atomicFail, 1)
	}

	// --- Mutex-protected path (slice appends, map updates, timeline) ---
	c.mu.Lock()

	c.allLatencies = append(c.allLatencies, rec.Duration)
	if rec.StatusCode > 0 {
		c.statusCodes[rec.StatusCode]++
	}
	if !rec.Success {
		errMsg := rec.ErrorMessage
		if errMsg == "" {
			errMsg = "unknown_error"
		}
		c.errors[errMsg]++
	}

	// Per-endpoint tracking
	if epColl, exists := c.endpoints[rec.EndpointName]; exists {
		atomic.AddInt64(&epColl.totalReqs, 1)
		if rec.Success {
			atomic.AddInt64(&epColl.successReqs, 1)
		} else {
			atomic.AddInt64(&epColl.failedReqs, 1)
			epColl.errors[rec.ErrorMessage]++
		}
		epColl.latencies = append(epColl.latencies, rec.Duration)
		if rec.StatusCode > 0 {
			epColl.statusCodes[rec.StatusCode]++
		}
	}

	// Timeline bucketing (per second)
	relSec := int64(time.Since(c.startTime).Seconds())
	if relSec < 0 {
		relSec = 0
	}
	if relSec > c.lastSec {
		c.lastSec = relSec
	}

	bucket, ok := c.timeline[relSec]
	if !ok {
		bucket = &types.TimeSeriesPoint{
			Timestamp:   rec.Timestamp,
			RelativeSec: float64(relSec),
		}
		c.timeline[relSec] = bucket
	}

	bucket.Requests++
	if rec.Success {
		bucket.Successes++
	} else {
		bucket.Failures++
	}
	c.bucketLatencies[relSec] = append(c.bucketLatencies[relSec], rec.Duration)

	c.mu.Unlock()
}

// RecordSystemSample attaches system resource metrics (CPU/RAM) to the current timeline point.
func (c *Collector) RecordSystemSample(cpuPercent float64, memoryMB float64, activeUsers int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	relSec := int64(time.Since(c.startTime).Seconds())
	if relSec < 0 {
		relSec = 0
	}
	bucket, ok := c.timeline[relSec]
	if !ok {
		bucket = &types.TimeSeriesPoint{
			Timestamp:   time.Now(),
			RelativeSec: float64(relSec),
		}
		c.timeline[relSec] = bucket
	}
	bucket.CPUPercent = cpuPercent
	bucket.MemoryMB = memoryMB
	bucket.ActiveUsers = activeUsers
}

// CurrentStats provides a real-time snapshot for the live progress display and circuit breaker.
// This method is fully lock-free via atomic reads — zero contention with workers.
func (c *Collector) CurrentStats() (total int64, success int64, fail int64, errorRate float64, avgLatency time.Duration) {
	total = atomic.LoadInt64(&c.atomicTotal)
	fail = atomic.LoadInt64(&c.atomicFail)
	success = atomic.LoadInt64(&c.atomicSuccess)

	if total > 0 {
		errorRate = (float64(fail) / float64(total)) * 100.0
		avgLatency = time.Duration(atomic.LoadInt64(&c.atomicLatencySum) / total)
	}
	return
}

// Stop marks the end of collection and computes final results.
func (c *Collector) Stop() types.TestResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.endTime = time.Now()
	totalDur := c.endTime.Sub(c.startTime)
	if totalDur <= 0 {
		totalDur = time.Millisecond
	}

	totalReqs := atomic.LoadInt64(&c.atomicTotal)
	failedReqs := atomic.LoadInt64(&c.atomicFail)
	successReqs := atomic.LoadInt64(&c.atomicSuccess)

	var errorRate float64
	if totalReqs > 0 {
		errorRate = (float64(failedReqs) / float64(totalReqs)) * 100.0
	}
	throughput := float64(totalReqs) / totalDur.Seconds()

	latencyStats := calculateLatencyStats(c.allLatencies)

	// Build endpoint results
	endpointResults := make([]types.EndpointResult, 0, len(c.endpoints))
	for _, epColl := range c.endpoints {
		epTotal := atomic.LoadInt64(&epColl.totalReqs)
		epFailed := atomic.LoadInt64(&epColl.failedReqs)
		epSuccess := atomic.LoadInt64(&epColl.successReqs)

		var epErrorRate float64
		if epTotal > 0 {
			epErrorRate = (float64(epFailed) / float64(epTotal)) * 100.0
		}
		epRPS := float64(epTotal) / totalDur.Seconds()

		endpointResults = append(endpointResults, types.EndpointResult{
			Name:            epColl.spec.Name,
			URL:             epColl.spec.URL,
			Method:          epColl.spec.Method,
			TotalRequests:   epTotal,
			SuccessRequests: epSuccess,
			FailedRequests:  epFailed,
			ErrorRate:       epErrorRate,
			RPS:             epRPS,
			Latency:         calculateLatencyStats(epColl.latencies),
			StatusCodes:     epColl.statusCodes,
			Errors:          epColl.errors,
		})
	}

	// Sort endpoints by name for deterministic output
	sort.Slice(endpointResults, func(i, j int) bool {
		return endpointResults[i].Name < endpointResults[j].Name
	})

	// Build timeline array sorted by relative second, with computed latencies
	timelineArray := make([]types.TimeSeriesPoint, 0, len(c.timeline))
	for sec := int64(0); sec <= c.lastSec; sec++ {
		if pt, ok := c.timeline[sec]; ok {
			pt.RPS = float64(pt.Requests)

			if lats, hasLats := c.bucketLatencies[sec]; hasLats && len(lats) > 0 {
				var latSum time.Duration
				for _, d := range lats {
					latSum += d
				}
				pt.AvgLatencyMs = float64(latSum.Microseconds()) / float64(len(lats)) / 1000.0

				bucketSorted := make([]time.Duration, len(lats))
				copy(bucketSorted, lats)
				sort.Slice(bucketSorted, func(i, j int) bool {
					return bucketSorted[i] < bucketSorted[j]
				})
				p95Idx := int(float64(len(bucketSorted)-1) * 0.95)
				pt.P95LatencyMs = float64(bucketSorted[p95Idx].Microseconds()) / 1000.0
			}

			timelineArray = append(timelineArray, *pt)
		}
	}

	return types.TestResult{
		StartTime:       c.startTime,
		EndTime:         c.endTime,
		Duration:        totalDur,
		TotalRequests:   totalReqs,
		SuccessRequests: successReqs,
		FailedRequests:  failedReqs,
		ErrorRate:       errorRate,
		ThroughputRPS:   throughput,
		Latency:         latencyStats,
		StatusCodes:     c.statusCodes,
		ErrorSummary:    c.errors,
		Endpoints:       endpointResults,
		Timeline:        timelineArray,
	}
}

// calculateLatencyStats sorts a slice of durations and computes percentiles.
func calculateLatencyStats(durations []time.Duration) types.LatencyStats {
	n := len(durations)
	if n == 0 {
		return types.LatencyStats{}
	}

	sorted := make([]time.Duration, n)
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	mean := sum / time.Duration(n)

	// Standard Deviation
	var varianceSum float64
	meanNs := float64(mean.Nanoseconds())
	for _, d := range sorted {
		diff := float64(d.Nanoseconds()) - meanNs
		varianceSum += diff * diff
	}
	stdDev := time.Duration(math.Sqrt(varianceSum / float64(n)))

	return types.LatencyStats{
		Min:    sorted[0],
		Max:    sorted[n-1],
		Mean:   mean,
		Median: percentile(sorted, 50),
		P90:    percentile(sorted, 90),
		P95:    percentile(sorted, 95),
		P99:    percentile(sorted, 99),
		P999:   percentile(sorted, 99.9),
		StdDev: stdDev,
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[n-1]
	}

	rank := (p / 100.0) * float64(n-1)
	low := int(math.Floor(rank))
	high := int(math.Ceil(rank))

	if low == high {
		return sorted[low]
	}

	weight := rank - float64(low)
	diff := float64(sorted[high] - sorted[low])
	return sorted[low] + time.Duration(diff*weight)
}
