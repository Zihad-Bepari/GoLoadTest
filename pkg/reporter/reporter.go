package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

// SaveResults writes JSON and HTML reports based on configuration.
func SaveResults(res types.TestResult, outputFormat string, outputDir string) error {
	if outputDir == "" {
		outputDir = "./results"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	baseName := fmt.Sprintf("loadtest-%s", timestamp)

	format := strings.ToLower(outputFormat)
	if format == "json" || format == "all" || format == "" {
		jsonPath := filepath.Join(outputDir, baseName+".json")
		if err := GenerateJSON(res, jsonPath); err != nil {
			return err
		}
	}

	if format == "html" || format == "all" || format == "" {
		htmlPath := filepath.Join(outputDir, baseName+".html")
		if err := GenerateHTML(res, htmlPath); err != nil {
			return err
		}
	}

	return nil
}

// GenerateJSON serializes test results into a formatted JSON file.
func GenerateJSON(res types.TestResult, filePath string) error {
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON results: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}

// GenerateHTML creates a standalone, self-contained dashboard report with interactive SVG charts.
func GenerateHTML(res types.TestResult, filePath string) error {
	htmlContent := renderHTML(res)
	return os.WriteFile(filePath, []byte(htmlContent), 0644)
}

func renderHTML(res types.TestResult) string {
	statusBadgeClass := "badge-pass"
	statusText := "PASSED"
	if res.Aborted {
		statusBadgeClass = "badge-abort"
		statusText = "ABORTED"
	} else if !res.Passed {
		statusBadgeClass = "badge-fail"
		statusText = "FAILED"
	}

	// Build SVG chart for RPS
	rpsChartSVG := generateLineSVG(res.Timeline, func(p types.TimeSeriesPoint) float64 { return p.RPS }, "#3b82f6", "RPS")

	// Build SVG chart for Latency
	latChartSVG := generateLineSVG(res.Timeline, func(p types.TimeSeriesPoint) float64 { return p.AvgLatencyMs }, "#10b981", "Avg Latency (ms)")

	// Build Threshold rows
	var thresholdRows strings.Builder
	for _, th := range res.Thresholds {
		passClass := "badge-pass"
		passText := "PASSED"
		if !th.Passed {
			passClass = "badge-fail"
			passText = "FAILED"
		}
		thresholdRows.WriteString(fmt.Sprintf(`
			<tr>
				<td><strong>%s</strong></td>
				<td><code>%s</code></td>
				<td><code>%s</code></td>
				<td><span class="badge %s">%s</span></td>
			</tr>`, th.Metric, th.Target, th.Actual, passClass, passText))
	}
	if len(res.Thresholds) == 0 {
		thresholdRows.WriteString(`<tr><td colspan="4" class="text-muted">No thresholds specified</td></tr>`)
	}

	// Build Bottleneck cards
	var bottleneckCards strings.Builder
	for _, b := range res.Bottlenecks {
		sevClass := "sev-info"
		switch b.Severity {
		case "CRITICAL":
			sevClass = "sev-critical"
		case "HIGH":
			sevClass = "sev-high"
		case "MEDIUM":
			sevClass = "sev-medium"
		}
		bottleneckCards.WriteString(fmt.Sprintf(`
			<div class="insight-card %s">
				<div class="insight-header">
					<span class="badge %s">%s</span>
					<h3>%s</h3>
				</div>
				<p class="insight-desc">%s</p>
				<p class="insight-evidence"><strong>Evidence:</strong> %s</p>
				<p class="insight-suggest"><strong>Suggestion:</strong> %s</p>
			</div>`, sevClass, sevClass, b.Severity, b.Title, b.Description, b.Evidence, b.Suggestion))
	}
	if len(res.Bottlenecks) == 0 {
		bottleneckCards.WriteString(`<div class="no-bottlenecks">🎉 No significant performance bottlenecks detected! The target system performed cleanly within target limits.</div>`)
	}

	// Build Endpoints table rows
	var endpointRows strings.Builder
	for _, ep := range res.Endpoints {
		var statusBadge string
		if ep.TotalRequests == 0 {
			statusBadge = `<span class="badge badge-abort">Not Reached (Server Down)</span>`
		} else if ep.ErrorRate >= 80 {
			statusBadge = fmt.Sprintf(`<span class="badge badge-fail">Crashed (%0.1f%% Error)</span>`, ep.ErrorRate)
		} else if ep.ErrorRate > 0 {
			statusBadge = fmt.Sprintf(`<span class="badge badge-abort">Degraded (%0.1f%% Error)</span>`, ep.ErrorRate)
		} else {
			statusBadge = `<span class="badge badge-pass">Stable &amp; Handled</span>`
		}

		userCap := "-"
		if ep.TotalRequests > 0 {
			if ep.BreakingUsers > 0 {
				userCap = fmt.Sprintf("Broke at %d Users", ep.BreakingUsers)
			} else if ep.MaxUsers > 0 {
				userCap = fmt.Sprintf("Handled %d Users", ep.MaxUsers)
			} else if res.BreakingPoint != nil && res.BreakingPoint.MaxStableConcurrency > 0 {
				userCap = fmt.Sprintf("Handled %d Users", res.BreakingPoint.MaxStableConcurrency)
			} else {
				userCap = "Stable"
			}
		}

		reqsDisplay := fmt.Sprintf("%d (Success: %d / Fail: %d)", ep.TotalRequests, ep.SuccessRequests, ep.FailedRequests)
		if ep.TotalRequests == 0 {
			reqsDisplay = "0 reqs (Server Unreachable)"
		}

		endpointRows.WriteString(fmt.Sprintf(`
			<tr>
				<td><span class="method-tag method-%s">%s</span> <strong>%s</strong><br/><small class="text-muted">%s</small></td>
				<td>%s</td>
				<td><strong>%s</strong></td>
				<td>%.1f req/s</td>
				<td>%v</td>
				<td>%v</td>
				<td>%s</td>
			</tr>`,
			strings.ToLower(ep.Method), ep.Method, ep.Name, ep.URL,
			reqsDisplay, userCap, ep.RPS, ep.Latency.Mean.Round(time.Millisecond), ep.Latency.P95.Round(time.Millisecond),
			statusBadge))
	}

	// Build status codes summary badges
	var statusBadges strings.Builder
	for code, count := range res.StatusCodes {
		badgeColor := "code-2xx"
		if code >= 300 && code < 400 {
			badgeColor = "code-3xx"
		} else if code >= 400 && code < 500 {
			badgeColor = "code-4xx"
		} else if code >= 500 {
			badgeColor = "code-5xx"
		}
		statusBadges.WriteString(fmt.Sprintf(`<span class="status-pill %s"><strong>%d:</strong> %d reqs</span> `, badgeColor, code, count))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>GoLoadTest Report - %s</title>
	<style>
		:root {
			--bg: #0f172a;
			--card-bg: #1e293b;
			--text: #f8fafc;
			--text-muted: #94a3b8;
			--border: #334155;
			--primary: #3b82f6;
			--success: #10b981;
			--danger: #ef4444;
			--warning: #f59e0b;
		}
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body {
			background: var(--bg);
			color: var(--text);
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			padding: 2rem;
			line-height: 1.5;
		}
		.container { max-width: 1200px; margin: 0 auto; }
		.header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			padding-bottom: 1.5rem;
			border-bottom: 1px solid var(--border);
			margin-bottom: 2rem;
		}
		.title-area h1 { font-size: 1.8rem; font-weight: 700; color: #fff; }
		.title-area p { color: var(--text-muted); font-size: 0.9rem; margin-top: 0.25rem; }
		.badge {
			padding: 0.35rem 0.8rem;
			border-radius: 9999px;
			font-size: 0.85rem;
			font-weight: 700;
			text-transform: uppercase;
			display: inline-block;
		}
		.badge-pass { background: rgba(16, 185, 129, 0.2); color: #34d399; border: 1px solid #10b981; }
		.badge-fail { background: rgba(239, 68, 68, 0.2); color: #f87171; border: 1px solid #ef4444; }
		.badge-abort { background: rgba(245, 158, 11, 0.2); color: #fbbf24; border: 1px solid #f59e0b; }
		.grid-cards {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 1rem;
			margin-bottom: 2rem;
		}
		.stat-card {
			background: var(--card-bg);
			border: 1px solid var(--border);
			border-radius: 0.75rem;
			padding: 1.25rem;
		}
		.stat-card .label { color: var(--text-muted); font-size: 0.8rem; font-weight: 600; text-transform: uppercase; }
		.stat-card .value { font-size: 1.75rem; font-weight: 700; margin-top: 0.25rem; }
		.charts-section {
			display: grid;
			grid-template-columns: 1fr 1fr;
			gap: 1.5rem;
			margin-bottom: 2rem;
		}
		@media (max-width: 850px) { .charts-section { grid-template-columns: 1fr; } }
		.chart-card {
			background: var(--card-bg);
			border: 1px solid var(--border);
			border-radius: 0.75rem;
			padding: 1.25rem;
		}
		.chart-card h3 { font-size: 1.1rem; margin-bottom: 1rem; color: #e2e8f0; }
		.table-card {
			background: var(--card-bg);
			border: 1px solid var(--border);
			border-radius: 0.75rem;
			padding: 1.25rem;
			margin-bottom: 2rem;
			overflow-x: auto;
		}
		.table-card h3 { font-size: 1.1rem; margin-bottom: 1rem; }
		table { width: 100%%; border-collapse: collapse; text-align: left; font-size: 0.9rem; }
		th { padding: 0.75rem; border-bottom: 1px solid var(--border); color: var(--text-muted); font-weight: 600; }
		td { padding: 0.75rem; border-bottom: 1px solid rgba(255, 255, 255, 0.05); }
		.method-tag {
			padding: 0.2rem 0.4rem;
			border-radius: 4px;
			font-size: 0.75rem;
			font-weight: 700;
			margin-right: 0.3rem;
		}
		.method-get { background: #0369a1; color: #bae6fd; }
		.method-post { background: #15803d; color: #bbf7d0; }
		.method-put { background: #b45309; color: #fde68a; }
		.method-delete { background: #b91c1c; color: #fecaca; }
		.insight-card {
			background: rgba(30, 41, 59, 0.7);
			border-radius: 0.5rem;
			padding: 1rem;
			margin-bottom: 1rem;
			border-left: 4px solid var(--primary);
		}
		.sev-critical { border-left-color: var(--danger); background: rgba(239, 68, 68, 0.05); }
		.sev-high { border-left-color: var(--warning); background: rgba(245, 158, 11, 0.05); }
		.sev-medium { border-left-color: var(--primary); }
		.insight-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.5rem; }
		.insight-header h3 { font-size: 1rem; }
		.insight-desc { font-size: 0.9rem; margin-bottom: 0.4rem; }
		.insight-evidence { font-size: 0.85rem; color: #94a3b8; margin-bottom: 0.3rem; }
		.insight-suggest { font-size: 0.85rem; color: #38bdf8; }
		.no-bottlenecks { padding: 1.5rem; text-align: center; color: #34d399; font-weight: 600; }
		.status-pill {
			display: inline-block;
			padding: 0.25rem 0.6rem;
			border-radius: 0.25rem;
			font-size: 0.8rem;
			margin: 0.2rem;
		}
		.code-2xx { background: rgba(16, 185, 129, 0.2); color: #34d399; }
		.code-3xx { background: rgba(59, 130, 246, 0.2); color: #60a5fa; }
		.code-4xx { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }
		.code-5xx { background: rgba(239, 68, 68, 0.2); color: #f87171; }
		.err-high { color: #f87171; font-weight: 700; }
		.err-ok { color: #34d399; }
		.text-muted { color: var(--text-muted); }
		svg { width: 100%%; height: 160px; overflow: visible; }
	</style>
</head>
<body>
	<div class="container">
		<header class="header">
			<div class="title-area">
				<h1>%s</h1>
				<p>Generated at %s &bull; Test Duration: %v</p>
			</div>
			<div>
				<span class="badge %s" style="font-size: 1.1rem; padding: 0.5rem 1.25rem;">%s</span>
			</div>
		</header>

		<!-- Key Metric Cards -->
		<div class="grid-cards">
			<div class="stat-card">
				<div class="label">Total Requests</div>
				<div class="value">%d</div>
			</div>
			<div class="stat-card">
				<div class="label">Throughput</div>
				<div class="value" style="color: #60a5fa;">%.1f <span style="font-size: 0.9rem;">RPS</span></div>
			</div>
			<div class="stat-card">
				<div class="label">Average Latency</div>
				<div class="value">%v</div>
			</div>
			<div class="stat-card">
				<div class="label">P95 Latency</div>
				<div class="value" style="color: #fbbf24;">%v</div>
			</div>
			<div class="stat-card">
				<div class="label">P99 Latency</div>
				<div class="value">%v</div>
			</div>
			<div class="stat-card">
				<div class="label">Error Rate</div>
				<div class="value %s">%.2f%%</div>
			</div>
			<div class="stat-card">
				<div class="label">Peak Host CPU</div>
				<div class="value">%.1f%%</div>
			</div>
			<div class="stat-card">
				<div class="label">Peak Memory</div>
				<div class="value">%.1f <span style="font-size: 0.9rem;">MB</span></div>
			</div>
		</div>

		<!-- Status Codes Overview -->
		<div style="margin-bottom: 2rem;">
			<h4 style="margin-bottom: 0.5rem; font-size: 0.9rem; text-transform: uppercase; color: var(--text-muted);">HTTP Status Codes</h4>
			<div>%s</div>
		</div>

		<!-- Interactive Charts -->
		<div class="charts-section">
			<div class="chart-card">
				<h3>Throughput Trend (Requests / Sec)</h3>
				%s
			</div>
			<div class="chart-card">
				<h3>Latency Trend (Avg Response Time)</h3>
				%s
			</div>
		</div>

		<!-- SLA Thresholds Validation -->
		<div class="table-card">
			<h3>SLA Performance Thresholds</h3>
			<table>
				<thead>
					<tr>
						<th>Metric</th>
						<th>Target SLA</th>
						<th>Actual Result</th>
						<th>Status</th>
					</tr>
				</thead>
				<tbody>
					%s
				</tbody>
			</table>
		</div>

		<!-- Automated Bottleneck Diagnostics -->
		<div class="table-card">
			<h3>Automated Bottleneck Insights & Diagnostics</h3>
			%s
		</div>

		<!-- Endpoint Breakdown -->
		<div class="table-card">
			<h3>Endpoint Breakdown &amp; Capacity</h3>
			<table>
				<thead>
					<tr>
						<th>Endpoint</th>
						<th>Handled Requests</th>
						<th>User Capacity</th>
						<th>Throughput (RPS)</th>
						<th>Avg Latency</th>
						<th>P95 Latency</th>
						<th>Status &amp; Breaking Point</th>
					</tr>
				</thead>
				<tbody>
					%s
				</tbody>
			</table>
		</div>

		<footer style="text-align: center; color: var(--text-muted); font-size: 0.8rem; padding: 2rem 0;">
			Generated by <strong>GoLoadTest</strong> &bull; Cross-Language API Performance &amp; System Testing Suite
		</footer>
	</div>
</body>
</html>`,
		res.SuiteName,
		res.SuiteName,
		res.EndTime.Format("2006-01-02 15:04:05"),
		res.Duration.Round(time.Millisecond),
		statusBadgeClass, statusText,
		res.TotalRequests,
		res.ThroughputRPS,
		res.Latency.Mean.Round(time.Microsecond),
		res.Latency.P95.Round(time.Microsecond),
		res.Latency.P99.Round(time.Microsecond),
		mapErrorClass(res.ErrorRate), res.ErrorRate,
		res.System.PeakCPUPercent,
		res.System.PeakMemoryMB,
		statusBadges.String(),
		rpsChartSVG,
		latChartSVG,
		thresholdRows.String(),
		bottleneckCards.String(),
		endpointRows.String(),
	)
}

func mapErrorClass(errRate float64) string {
	if errRate > 2.0 {
		return "err-high"
	}
	return "err-ok"
}

func generateLineSVG(timeline []types.TimeSeriesPoint, valueExtractor func(types.TimeSeriesPoint) float64, color string, label string) string {
	if len(timeline) == 0 {
		return `<div style="text-align:center; padding: 40px; color: #94a3b8;">No timeline data points recorded.</div>`
	}

	width := 500.0
	height := 140.0
	padding := 30.0

	var maxVal float64 = 1.0
	for _, p := range timeline {
		v := valueExtractor(p)
		if v > maxVal {
			maxVal = v
		}
	}

	numPoints := len(timeline)
	stepX := (width - 2*padding) / float64(numPoints-1)
	if numPoints == 1 {
		stepX = width - 2*padding
	}

	var points []string
	for i, p := range timeline {
		v := valueExtractor(p)
		x := padding + float64(i)*stepX
		y := height - padding - (v/maxVal)*(height-2*padding)
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
	}

	polyline := strings.Join(points, " ")

	return fmt.Sprintf(`
		<svg viewBox="0 0 500 160">
			<!-- Grid lines -->
			<line x1="30" y1="20" x2="480" y2="20" stroke="#334155" stroke-dasharray="3,3" />
			<line x1="30" y1="75" x2="480" y2="75" stroke="#334155" stroke-dasharray="3,3" />
			<line x1="30" y1="130" x2="480" y2="130" stroke="#475569" />
			<!-- Y-Axis labels -->
			<text x="5" y="25" fill="#94a3b8" font-size="10">%.1f</text>
			<text x="5" y="130" fill="#94a3b8" font-size="10">0</text>
			<!-- Sparkline Polyline -->
			<polyline fill="none" stroke="%s" stroke-width="2.5" points="%s" />
		</svg>`,
		maxVal, color, polyline)
}
