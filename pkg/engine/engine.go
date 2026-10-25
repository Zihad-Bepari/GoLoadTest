package engine

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goloadtest/goloadtest/pkg/metrics"
	"github.com/goloadtest/goloadtest/pkg/monitor"
	"github.com/goloadtest/goloadtest/pkg/requester"
	"github.com/goloadtest/goloadtest/pkg/types"
)

// ProgressSnapshot provides real-time state for live CLI UI or listeners.
type ProgressSnapshot struct {
	Elapsed        time.Duration
	TotalDuration  time.Duration
	TotalRequests  int64
	SuccessReqs    int64
	FailedReqs     int64
	CurrentRPS     float64
	CurrentAvgLat  time.Duration
	ErrorRate      float64
	ActiveWorkers  int
	CircuitTripped bool
	AbortReason    string
}

// Engine drives the execution of concurrent load generation.
type Engine struct {
	cfg        types.Config
	client     *requester.Client
	collector  *metrics.Collector
	onProgress func(ProgressSnapshot)

	// Runtime state
	cancelFunc       context.CancelFunc
	activeWorkers    int64
	desiredWorkers   int64 // atomic: current target worker count for dynamic patterns
	consecutiveFails int64
	aborted          int32 // atomic bool
	abortReason      string
	abortMu          sync.Mutex
	breakingPoint    *types.BreakingPointInfo
}

// New creates a new load testing engine.
func New(cfg types.Config) *Engine {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}

	client := requester.NewClient(15*time.Second, cfg.Concurrency*2, cfg.AllowInsecureTLS)
	collector := metrics.NewCollector(cfg.Endpoints)

	return &Engine{
		cfg:       cfg,
		client:    client,
		collector: collector,
	}
}

// SetProgressListener sets a callback invoked periodically with live stats.
func (e *Engine) SetProgressListener(cb func(ProgressSnapshot)) {
	e.onProgress = cb
}

// Collector returns the internal metrics collector.
func (e *Engine) Collector() *metrics.Collector {
	return e.collector
}

// Run executes the load test according to configuration.
func (e *Engine) Run(parentCtx context.Context) (types.TestResult, error) {
	if len(e.cfg.Endpoints) == 0 {
		return types.TestResult{}, fmt.Errorf("no endpoints specified for load test")
	}

	ctx, cancel := context.WithTimeout(parentCtx, e.cfg.Duration+5*time.Second)
	e.cancelFunc = cancel
	defer cancel()

	e.collector.Start()
	startTime := time.Now()

	// Rate limiter channel if TargetRPS is set
	var rpsTicker *time.Ticker
	var rpsCh chan struct{}
	if e.cfg.TargetRPS > 0 {
		interval := time.Duration(float64(time.Second) / e.cfg.TargetRPS)
		if interval < time.Microsecond {
			interval = time.Microsecond
		}
		rpsTicker = time.NewTicker(interval)
		defer rpsTicker.Stop()
		rpsCh = make(chan struct{}, int(e.cfg.TargetRPS)+10)

		// Feeder goroutine
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-rpsTicker.C:
					select {
					case rpsCh <- struct{}{}:
					default:
					}
				}
			}
		}()
	}

	// Prepare weighted endpoints selector
	selector := newEndpointSelector(e.cfg.Endpoints)

	// Progress monitor goroutine
	progressDone := make(chan struct{})
	go e.runProgressMonitor(ctx, startTime, progressDone)

	// Concurrency Controller / Worker Dispatcher
	var wg sync.WaitGroup
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	maxWorkers := e.cfg.Concurrency
	atomic.StoreInt64(&e.desiredWorkers, int64(maxWorkers))

	// Worker supervisor: periodically updates desiredWorkers atomically
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-workerCtx.Done():
				return
			case now := <-ticker.C:
				elapsed := now.Sub(startTime)
				if elapsed >= e.cfg.Duration {
					cancelWorkers()
					return
				}

				desired := e.calculateDesiredWorkers(elapsed, maxWorkers)
				atomic.StoreInt64(&e.desiredWorkers, int64(desired))
			}
		}
	}()

	// Spawn initial pool of workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()
			e.runWorker(workerCtx, workerID, selector, rpsCh)
		}()
	}

	// Start system monitor if enabled
	mon := monitor.New(e.cfg.Monitor, e.collector)
	mon.Start(ctx)

	// Wait for workers or timeout
	wg.Wait()
	close(progressDone)

	// Finalize metrics
	result := e.collector.Stop()
	result.SuiteName = e.cfg.Name
	result.System = mon.Stop()

	if atomic.LoadInt32(&e.aborted) == 1 {
		result.Aborted = true
		e.abortMu.Lock()
		result.AbortReason = e.abortReason
		e.abortMu.Unlock()
	}

	e.abortMu.Lock()
	if e.breakingPoint != nil {
		e.breakingPoint.CrashRPS = result.ThroughputRPS
		e.breakingPoint.MaxStableRPS = result.ThroughputRPS * 0.9
		result.BreakingPoint = e.breakingPoint
	}
	e.abortMu.Unlock()

	return result, nil
}

func (e *Engine) calculateDesiredWorkers(elapsed time.Duration, maxWorkers int) int {
	totalDur := e.cfg.Duration
	rampUp := e.cfg.RampUp
	rampDown := e.cfg.RampDown

	if rampUp+rampDown > totalDur {
		rampUp = totalDur / 2
		rampDown = totalDur / 2
	}

	switch e.cfg.Pattern {
	case types.PatternRamp:
		if elapsed < rampUp && rampUp > 0 {
			fraction := float64(elapsed) / float64(rampUp)
			workers := int(fraction * float64(maxWorkers))
			if workers < 1 {
				return 1
			}
			return workers
		} else if elapsed > (totalDur-rampDown) && rampDown > 0 {
			remaining := totalDur - elapsed
			fraction := float64(remaining) / float64(rampDown)
			workers := int(fraction * float64(maxWorkers))
			if workers < 1 {
				return 1
			}
			return workers
		}
		return maxWorkers

	case types.PatternSpike:
		// Normal load (20%), spikes to 100% in middle third, then returns to 20%
		oneThird := totalDur / 3
		twoThirds := 2 * totalDur / 3
		if elapsed >= oneThird && elapsed <= twoThirds {
			return maxWorkers
		}
		base := maxWorkers / 5
		if base < 1 {
			base = 1
		}
		return base

	case types.PatternStep:
		// Steps in 4 increments: 25%, 50%, 75%, 100%
		stepDur := totalDur / 4
		stepIndex := int(elapsed / stepDur)
		if stepIndex > 3 {
			stepIndex = 3
		}
		workers := int(float64(maxWorkers) * float64(stepIndex+1) / 4.0)
		if workers < 1 {
			return 1
		}
		return workers

	case types.PatternBreakToCrash:
		stress := e.cfg.Stress
		initial := stress.InitialConcurrency
		if initial <= 0 {
			initial = 5
		}
		step := stress.StepConcurrency
		if step <= 0 {
			step = 10
		}
		interval := stress.StepInterval
		if interval <= 0 {
			interval = 2 * time.Second
		}
		stepCount := int(elapsed / interval)
		targetWorkers := initial + (stepCount * step)
		if stress.MaxConcurrency > 0 && targetWorkers > stress.MaxConcurrency {
			targetWorkers = stress.MaxConcurrency
		}
		if targetWorkers > maxWorkers {
			targetWorkers = maxWorkers
		}
		if targetWorkers < 1 {
			return 1
		}
		return targetWorkers

	default: // PatternConstant
		if rampUp > 0 && elapsed < rampUp {
			fraction := float64(elapsed) / float64(rampUp)
			workers := int(fraction * float64(maxWorkers))
			if workers < 1 {
				return 1
			}
			return workers
		}
		return maxWorkers
	}
}

func (e *Engine) runWorker(
	ctx context.Context,
	workerID int,
	selector *endpointSelector,
	rpsCh chan struct{},
) {
	atomic.AddInt64(&e.activeWorkers, 1)
	defer atomic.AddInt64(&e.activeWorkers, -1)

	for {
		// Check if aborted
		if atomic.LoadInt32(&e.aborted) == 1 {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		// If this worker's index is currently throttled off by ramp, idle briefly
		currentActiveLimit := int(atomic.LoadInt64(&e.desiredWorkers))
		if workerID >= currentActiveLimit {
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
				continue
			}
		}

		// Enforce RPS if rate limiter is active
		if rpsCh != nil {
			select {
			case <-ctx.Done():
				return
			case <-rpsCh:
			}
		}

		// Select endpoint
		spec := selector.Next()

		// Execute request
		record := e.client.Execute(ctx, spec)

		// If context was cancelled during execution, do not penalize metrics
		if ctx.Err() != nil && record.ErrorMessage == "context_canceled" {
			return
		}

		// Record metrics
		e.collector.Record(record)

		// Check circuit breaker or crash condition
		if e.cfg.Pattern == types.PatternBreakToCrash || e.cfg.Stress.Enabled {
			e.checkStressCrashCondition(record)
		} else {
			e.checkCircuitBreaker(record)
		}
	}
}

func (e *Engine) checkStressCrashCondition(rec requester.ResponseRecord) {
	if !rec.Success {
		atomic.AddInt64(&e.consecutiveFails, 1)
	} else {
		atomic.StoreInt64(&e.consecutiveFails, 0)
	}

	stress := e.cfg.Stress
	crashFails := stress.CrashConsecutiveFails
	if crashFails <= 0 {
		crashFails = 25
	}
	crashErrorRate := stress.CrashErrorRate
	if crashErrorRate <= 0 {
		crashErrorRate = 80.0
	}
	crashLatency := stress.CrashMaxLatency
	if crashLatency <= 0 {
		crashLatency = 15 * time.Second
	}

	fails := atomic.LoadInt64(&e.consecutiveFails)
	total, _, _, errRate, avgLat := e.collector.CurrentStats()
	active := int(atomic.LoadInt64(&e.desiredWorkers))

	var reason string
	if fails >= int64(crashFails) {
		reason = fmt.Sprintf("Server crashed or unreachable: %d consecutive failed/dropped requests", fails)
	} else if total >= 30 && errRate >= crashErrorRate {
		reason = fmt.Sprintf("Server failure threshold reached: Error rate at %.1f%% (threshold %.1f%%)", errRate, crashErrorRate)
	} else if total >= 30 && avgLat >= crashLatency {
		reason = fmt.Sprintf("Server freeze detected: Average latency at %v (threshold %v)", avgLat, crashLatency)
	}

	if reason != "" {
		e.abortMu.Lock()
		if e.breakingPoint == nil {
			stableCon := active - stress.StepConcurrency
			if stableCon < 1 {
				stableCon = 1
			}
			e.breakingPoint = &types.BreakingPointInfo{
				Detected:             true,
				CrashConcurrency:     active,
				MaxStableConcurrency: stableCon,
				CrashLatency:         avgLat,
				CrashErrorRate:       errRate,
				CrashReason:          reason,
			}
		}
		e.abortMu.Unlock()

		e.triggerAbort(fmt.Sprintf("⚡ Breaking Point Reached: %s", reason))
	}
}

func (e *Engine) checkCircuitBreaker(rec requester.ResponseRecord) {
	if !e.cfg.CircuitBreaker.Enabled {
		return
	}

	if !rec.Success {
		atomic.AddInt64(&e.consecutiveFails, 1)
	} else {
		atomic.StoreInt64(&e.consecutiveFails, 0)
	}

	// 1. Consecutive fails trip
	if e.cfg.CircuitBreaker.ConsecutiveFails > 0 {
		fails := atomic.LoadInt64(&e.consecutiveFails)
		if fails >= int64(e.cfg.CircuitBreaker.ConsecutiveFails) {
			e.triggerAbort(fmt.Sprintf("Circuit Breaker Tripped: %d consecutive failed requests", fails))
			return
		}
	}

	// 2. Error rate & Latency check (after minimum 20 requests sample)
	total, _, _, errRate, avgLat := e.collector.CurrentStats()
	if total >= 20 {
		if e.cfg.CircuitBreaker.MaxErrorRate > 0 && errRate >= e.cfg.CircuitBreaker.MaxErrorRate {
			e.triggerAbort(fmt.Sprintf("Circuit Breaker Tripped: Error rate reached %.1f%% (max: %.1f%%)", errRate, e.cfg.CircuitBreaker.MaxErrorRate))
			return
		}
		if e.cfg.CircuitBreaker.MaxLatency > 0 && avgLat >= e.cfg.CircuitBreaker.MaxLatency {
			e.triggerAbort(fmt.Sprintf("Circuit Breaker Tripped: Average latency reached %v (max: %v)", avgLat, e.cfg.CircuitBreaker.MaxLatency))
			return
		}
	}
}

func (e *Engine) triggerAbort(reason string) {
	if atomic.CompareAndSwapInt32(&e.aborted, 0, 1) {
		e.abortMu.Lock()
		e.abortReason = reason
		e.abortMu.Unlock()
		if e.cancelFunc != nil {
			e.cancelFunc()
		}
	}
}

func (e *Engine) runProgressMonitor(ctx context.Context, startTime time.Time, done chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastReqs int64
	var lastTime = startTime

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			total, success, fail, errRate, avgLat := e.collector.CurrentStats()

			timeDelta := now.Sub(lastTime).Seconds()
			var curRPS float64
			if timeDelta > 0 {
				curRPS = float64(total-lastReqs) / timeDelta
			}
			lastReqs = total
			lastTime = now

			active := int(atomic.LoadInt64(&e.activeWorkers))
			isAborted := atomic.LoadInt32(&e.aborted) == 1

			e.abortMu.Lock()
			reason := e.abortReason
			e.abortMu.Unlock()

			if e.onProgress != nil {
				e.onProgress(ProgressSnapshot{
					Elapsed:        now.Sub(startTime),
					TotalDuration:  e.cfg.Duration,
					TotalRequests:  total,
					SuccessReqs:    success,
					FailedReqs:     fail,
					CurrentRPS:     curRPS,
					CurrentAvgLat:  avgLat,
					ErrorRate:      errRate,
					ActiveWorkers:  active,
					CircuitTripped: isAborted,
					AbortReason:    reason,
				})
			}
		}
	}
}

// endpointSelector distributes requests according to weights.
type endpointSelector struct {
	pool []types.EndpointSpec
	idx  uint64
}

func newEndpointSelector(specs []types.EndpointSpec) *endpointSelector {
	var pool []types.EndpointSpec
	for _, spec := range specs {
		weight := spec.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			pool = append(pool, spec)
		}
	}
	if len(pool) == 0 {
		pool = specs
	}

	// Shuffle for better distribution
	rand.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})

	return &endpointSelector{pool: pool}
}

func (s *endpointSelector) Next() types.EndpointSpec {
	idx := atomic.AddUint64(&s.idx, 1) - 1
	return s.pool[idx%uint64(len(s.pool))]
}
