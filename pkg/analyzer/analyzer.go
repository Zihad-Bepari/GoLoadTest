package analyzer

import (
	"fmt"
	"time"

	"github.com/goloadtest/goloadtest/pkg/types"
)

// Analyze processes a TestResult, evaluates thresholds, and generates diagnostic bottleneck insights.
func Analyze(cfg types.Config, res *types.TestResult) {
	evaluateThresholds(cfg.Thresholds, res)
	detectBottlenecks(res)
}

func evaluateThresholds(th types.Thresholds, res *types.TestResult) {
	evals := make([]types.ThresholdEvaluation, 0)
	allPassed := true

	// Check P95 Latency
	if th.MaxP95 > 0 {
		passed := res.Latency.P95 <= th.MaxP95
		if !passed {
			allPassed = false
		}
		evals = append(evals, types.ThresholdEvaluation{
			Metric:   "P95 Latency",
			Target:   fmt.Sprintf("<= %v", th.MaxP95),
			Actual:   fmt.Sprintf("%v", res.Latency.P95),
			Passed:   passed,
			Severity: "CRITICAL",
		})
	}

	// Check P99 Latency
	if th.MaxP99 > 0 {
		passed := res.Latency.P99 <= th.MaxP99
		if !passed {
			allPassed = false
		}
		evals = append(evals, types.ThresholdEvaluation{
			Metric:   "P99 Latency",
			Target:   fmt.Sprintf("<= %v", th.MaxP99),
			Actual:   fmt.Sprintf("%v", res.Latency.P99),
			Passed:   passed,
			Severity: "HIGH",
		})
	}

	// Check Average Latency
	if th.MaxAvgLatency > 0 {
		passed := res.Latency.Mean <= th.MaxAvgLatency
		if !passed {
			allPassed = false
		}
		evals = append(evals, types.ThresholdEvaluation{
			Metric:   "Average Latency",
			Target:   fmt.Sprintf("<= %v", th.MaxAvgLatency),
			Actual:   fmt.Sprintf("%v", res.Latency.Mean),
			Passed:   passed,
			Severity: "MEDIUM",
		})
	}

	// Check Error Rate
	if th.MaxErrorRate >= 0 {
		passed := res.ErrorRate <= th.MaxErrorRate
		if !passed {
			allPassed = false
		}
		evals = append(evals, types.ThresholdEvaluation{
			Metric:   "Error Rate",
			Target:   fmt.Sprintf("<= %.2f%%", th.MaxErrorRate),
			Actual:   fmt.Sprintf("%.2f%%", res.ErrorRate),
			Passed:   passed,
			Severity: "CRITICAL",
		})
	}

	// Check Min Throughput
	if th.MinThroughput > 0 {
		passed := res.ThroughputRPS >= th.MinThroughput
		if !passed {
			allPassed = false
		}
		evals = append(evals, types.ThresholdEvaluation{
			Metric:   "Throughput (RPS)",
			Target:   fmt.Sprintf(">= %.1f RPS", th.MinThroughput),
			Actual:   fmt.Sprintf("%.1f RPS", res.ThroughputRPS),
			Passed:   passed,
			Severity: "HIGH",
		})
	}

	if res.Aborted {
		allPassed = false
	}

	res.Thresholds = evals
	res.Passed = allPassed
}

func detectBottlenecks(res *types.TestResult) {
	insights := make([]types.BottleneckInsight, 0)

	// 0. Breaking Point Analysis (Crash until breakdown)
	if res.BreakingPoint != nil && res.BreakingPoint.Detected {
		insights = append(insights, types.BottleneckInsight{
			Title:       "⚡ Server Breaking Point Identified",
			Severity:    "CRITICAL",
			Description: fmt.Sprintf("Under stress ramp-up, the server crashed / became unresponsive at concurrency %d (Last stable concurrency: ~%d).", res.BreakingPoint.CrashConcurrency, res.BreakingPoint.MaxStableConcurrency),
			Evidence:    fmt.Sprintf("Cause: %s | Peak Latency: %v | Error Rate: %.1f%%", res.BreakingPoint.CrashReason, res.BreakingPoint.CrashLatency, res.BreakingPoint.CrashErrorRate),
			Suggestion:  "Review thread pool capacity, DB connection pooling limits, socket backlog (somaxconn), and memory footprint before scaling.",
		})
	}

	// 1. Check for high error rate
	if res.ErrorRate >= 2.0 {
		insights = append(insights, types.BottleneckInsight{
			Title:       "Elevated Error Rate Detected",
			Severity:    "CRITICAL",
			Description: fmt.Sprintf("Test observed an overall error rate of %.2f%%, which exceeds standard reliability targets.", res.ErrorRate),
			Evidence:    fmt.Sprintf("%d out of %d requests failed.", res.FailedRequests, res.TotalRequests),
			Suggestion:  "Inspect server logs for unhandled 5xx exceptions or network timeouts during load.",
		})
	}

	// 2. Check for P95 latency degradation
	if res.Latency.P95 > 100*time.Millisecond {
		insights = append(insights, types.BottleneckInsight{
			Title:       "High P95 Latency Detected",
			Severity:    "HIGH",
			Description: fmt.Sprintf("P95 latency reached %v, which may degrade end-user experience under load.", res.Latency.P95),
			Evidence:    fmt.Sprintf("P95: %v (Mean: %v, Median: %v).", res.Latency.P95, res.Latency.Mean, res.Latency.Median),
			Suggestion:  "Analyze slow database queries, missing database indexes, or optimize backend computation.",
		})
	}

	// 2. Check for P99/P50 tail latency ratio (tail latency blowout)
	if res.Latency.Median > 0 && res.Latency.P99 > (res.Latency.Median*5) && res.Latency.P99 > 200*time.Millisecond {
		insights = append(insights, types.BottleneckInsight{
			Title:       "High Tail Latency (P99 Outlier Disparity)",
			Severity:    "HIGH",
			Description: "P99 latency is more than 5x the median response time, indicating significant tail outliers.",
			Evidence:    fmt.Sprintf("Median: %v vs P99: %v (Ratio: %.1fx).", res.Latency.Median, res.Latency.P99, float64(res.Latency.P99)/float64(res.Latency.Median)),
			Suggestion:  "Check for garbage collection pauses, database query lock contention, or slow external third-party calls.",
		})
	}

	// 3. Endpoint Comparison: identify outlier slow endpoints
	if len(res.Endpoints) > 1 {
		var slowestName string
		var slowestP95 time.Duration
		var fastestP95 time.Duration = time.Hour

		for _, ep := range res.Endpoints {
			if ep.Latency.P95 > slowestP95 {
				slowestP95 = ep.Latency.P95
				slowestName = ep.Name
			}
			if ep.Latency.P95 > 0 && ep.Latency.P95 < fastestP95 {
				fastestP95 = ep.Latency.P95
			}
		}

		if slowestP95 > (fastestP95*3) && slowestP95 > 150*time.Millisecond {
			insights = append(insights, types.BottleneckInsight{
				Title:       fmt.Sprintf("Endpoint Bottleneck: '%s'", slowestName),
				Severity:    "HIGH",
				Description: fmt.Sprintf("Endpoint '%s' is significantly slower than other endpoints in the suite.", slowestName),
				Evidence:    fmt.Sprintf("P95 latency is %v (compared to fastest endpoint at %v).", slowestP95, fastestP95),
				Suggestion:  "Profile the controller and database queries specific to this endpoint. Consider indexing or caching.",
			})
		}
	}

	// 4. Rate Limiting / 429 Throttle
	if count, ok := res.StatusCodes[429]; ok && count > 0 {
		insights = append(insights, types.BottleneckInsight{
			Title:       "Rate Limiting Encountered (HTTP 429)",
			Severity:    "MEDIUM",
			Description: "The target application returned HTTP 429 Too Many Requests responses.",
			Evidence:    fmt.Sprintf("%d requests were throttled by rate limiters.", count),
			Suggestion:  "Check API rate limiter thresholds, reverse proxy (Nginx/Cloudflare) rules, or distributed token buckets.",
		})
	}

	// 5. Connection refused / Gateway timeouts
	if count, ok := res.StatusCodes[502]; ok && count > 0 {
		insights = append(insights, types.BottleneckInsight{
			Title:       "Upstream Bad Gateway (HTTP 502)",
			Severity:    "CRITICAL",
			Description: "Upstream server or reverse proxy dropped backend connection.",
			Evidence:    fmt.Sprintf("%d requests failed with HTTP 502.", count),
			Suggestion:  "Verify backend process worker limits (e.g. PHP-FPM max_children, Node.js cluster size, or Gin max connections).",
		})
	}

	// 6. High CPU or Memory usage
	if res.System.PeakCPUPercent > 85.0 {
		insights = append(insights, types.BottleneckInsight{
			Title:       "CPU Saturation Observed",
			Severity:    "HIGH",
			Description: fmt.Sprintf("Host CPU utilization peaked at %.1f%%.", res.System.PeakCPUPercent),
			Evidence:    fmt.Sprintf("Peak CPU: %.1f%% (Average: %.1f%%).", res.System.PeakCPUPercent, res.System.AverageCPUPercent),
			Suggestion:  "Application or test host reached CPU limits. Consider optimizing heavy computation or scaling horizontally.",
		})
	}

	res.Bottlenecks = insights
}

// RegressionReport compares a current test result against a baseline result.
type RegressionReport struct {
	ThroughputChangePct float64 `json:"throughputChangePct"`
	P95LatencyChangePct float64 `json:"p95LatencyChangePct"`
	ErrorRateChangePct  float64 `json:"errorRateChangePct"`
	Regressed           bool    `json:"regressed"`
	Summary             string  `json:"summary"`
}

// CompareBaseline calculates differences between the current run and a saved baseline.
func CompareBaseline(current, baseline types.TestResult) RegressionReport {
	var tpPct, p95Pct, errPct float64

	if baseline.ThroughputRPS > 0 {
		tpPct = ((current.ThroughputRPS - baseline.ThroughputRPS) / baseline.ThroughputRPS) * 100.0
	}

	if baseline.Latency.P95 > 0 {
		p95Pct = ((float64(current.Latency.P95) - float64(baseline.Latency.P95)) / float64(baseline.Latency.P95)) * 100.0
	}

	errPct = current.ErrorRate - baseline.ErrorRate

	regressed := p95Pct > 15.0 || tpPct < -15.0 || errPct > 2.0

	summary := fmt.Sprintf("Throughput: %.1f%%, P95 Latency: %+.1f%%, Error Rate: %+.2f%%", tpPct, p95Pct, errPct)
	if regressed {
		summary = "REGRESSION DETECTED: " + summary
	} else {
		summary = "PERFORMANCE STABLE: " + summary
	}

	return RegressionReport{
		ThroughputChangePct: tpPct,
		P95LatencyChangePct: p95Pct,
		ErrorRateChangePct:  errPct,
		Regressed:           regressed,
		Summary:             summary,
	}
}
