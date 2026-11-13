package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest/pkg/analyzer"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/engine"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/reporter"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/server"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

var version = "1.0.0"

func init() {
	// Guard against OS thread exhaustion on shared hosting / containerized environments (cPanel, CloudLinux, Docker, etc.)
	// Configurable via GLT_MAX_THREADS and standard GOMAXPROCS environment variables.
	if maxThreadsStr := os.Getenv("GLT_MAX_THREADS"); maxThreadsStr != "" {
		if t, err := strconv.Atoi(maxThreadsStr); err == nil && t > 0 {
			debug.SetMaxThreads(t)
		}
	} else if os.Getenv("CPANEL") != "" || os.Getenv("SHARED_HOSTING") != "" {
		debug.SetMaxThreads(45)
	}

	if os.Getenv("GOMAXPROCS") == "" && (os.Getenv("CPANEL") != "" || os.Getenv("SHARED_HOSTING") != "") {
		if runtime.NumCPU() > 2 {
			runtime.GOMAXPROCS(2)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]
	switch command {
	case "version", "-v", "--version":
		fmt.Printf("GoLoadTest v%s - Cross-Language API Performance Suite\n", version)
		os.Exit(0)

	case "init":
		initConfigFile()
		os.Exit(0)

	case "run":
		runCmd(os.Args[2:])

	case "ui", "serve", "studio":
		uiCmd(os.Args[2:])

	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)

	default:
		// If first arg looks like flags or URL, treat as run
		if strings.HasPrefix(command, "-") || strings.HasPrefix(command, "http") {
			runCmd(os.Args[1:])
		} else {
			fmt.Printf("Unknown command '%s'. Run 'goloadtest help' for usage.\n", command)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Printf(`GoLoadTest v%s - High-Performance Cross-Language Load Testing Engine

Usage:
  goloadtest [command] [options]

Commands:
  run       Execute a load test suite
  ui        Launch the interactive real-time web dashboard & test runner
  init      Generate a starter configuration file (loadtest.json)
  version   Display version information
  help      Display help information

Run Options:
  --config string       Path to test configuration JSON file
  --url string          Target URL (shortcut mode)
  --method string       HTTP method (default: GET)
  --duration string     Test duration, e.g. 10s, 1m (default: 10s)
  --concurrency int     Number of concurrent workers (default: 10)
  --rps float           Target requests per second limit (0 = unthrottled)
  --output string       Output directory for reports (default: ./results)
  --format string       Output format: html, json, all, none (default: all)
  --max-p95 string      SLA max P95 latency threshold, e.g. 200ms
  --max-error float     SLA max error percentage threshold, e.g. 1.0 (1%%)

Examples:
  goloadtest run --config loadtest.json
  goloadtest run --url http://localhost:8080/api/users --duration 30s --concurrency 50
  goloadtest init
`, version)
}

func uiCmd(args []string) {
	fs := flag.NewFlagSet("ui", flag.ExitOnError)
	port := fs.Int("port", 8989, "Port to listen on")
	host := fs.String("host", "127.0.0.1", "Host binding address")
	_ = fs.Parse(args)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	srv := server.NewServer(addr)
	if err := srv.Start(); err != nil {
		fmt.Printf("Error starting dashboard server: %v\n", err)
		os.Exit(1)
	}
}

func initConfigFile() {
	fileName := "loadtest.json"
	if _, err := os.Stat(fileName); err == nil {
		fmt.Printf("Configuration file '%s' already exists.\n", fileName)
		return
	}

	rawJSON := `{
  "name": "API Load Test Suite",
  "endpoints": [
    {
      "name": "Get Users",
      "url": "http://localhost:8080/api/users",
      "method": "GET",
      "timeout": "5s",
      "expectedStatus": [200],
      "weight": 3
    },
    {
      "name": "Create User",
      "url": "http://localhost:8080/api/users",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": "{\"name\":\"John Doe\",\"email\":\"john@example.com\"}",
      "timeout": "5s",
      "expectedStatus": [200, 201],
      "weight": 1
    }
  ],
  "duration": "10s",
  "concurrency": 10,
  "rampUp": "2s",
  "rampDown": "1s",
  "targetRps": 0,
  "pattern": "constant",
  "circuitBreaker": {
    "enabled": true,
    "maxErrorRate": 50.0,
    "consecutiveFails": 100
  },
  "thresholds": {
    "maxP95": "300ms",
    "maxP99": "1s",
    "maxAvgLatency": "150ms",
    "maxErrorRate": 2.0,
    "minThroughput": 10.0
  },
  "monitor": {
    "enabled": true,
    "localHost": true
  },
  "outputFormat": "all",
  "outputPath": "./results"
}
`
	if err := os.WriteFile(fileName, []byte(rawJSON), 0644); err != nil {
		fmt.Printf("Failed to write file '%s': %v\n", fileName, err)
		os.Exit(1)
	}

	fmt.Printf("Created starter configuration file '%s'.\nRun with: goloadtest run --config %s\n", fileName, fileName)
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)

	configPath := fs.String("config", "", "Path to configuration JSON file")
	targetURL := fs.String("url", "", "Target URL")
	method := fs.String("method", "GET", "HTTP method")
	durationStr := fs.String("duration", "10s", "Test duration")
	concurrency := fs.Int("concurrency", 10, "Concurrent workers")
	rps := fs.Float64("rps", 0, "Target RPS limit")
	outputDir := fs.String("output", "./results", "Output directory for reports")
	outputFormat := fs.String("format", "all", "Output format (json, html, all, none)")
	maxP95Str := fs.String("max-p95", "", "SLA max P95 latency (e.g. 200ms)")
	maxErrorRate := fs.Float64("max-error", -1, "SLA max error rate percentage")
	breakToCrash := fs.Bool("break-to-crash", false, "Ramp pressure continuously until target crashes")
	stepVU := fs.Int("step-vu", 20, "Virtual user increment for break-to-crash mode")

	_ = fs.Parse(args)

	var cfg types.Config

	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			fmt.Printf("Error reading config file '%s': %v\n", *configPath, err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			fmt.Printf("Error parsing JSON config '%s': %v\n", *configPath, err)
			os.Exit(1)
		}
	} else if *targetURL != "" {
		dur, err := time.ParseDuration(*durationStr)
		if err != nil {
			dur = 10 * time.Second
		}

		cfg = types.DefaultConfig()
		cfg.Duration = dur
		cfg.Concurrency = *concurrency
		cfg.TargetRPS = *rps
		cfg.OutputPath = *outputDir
		cfg.OutputFormat = *outputFormat
		cfg.Endpoints = []types.EndpointSpec{
			{
				Name:   "Target Endpoint",
				URL:    *targetURL,
				Method: *method,
			},
		}

		if *maxP95Str != "" {
			if p95Dur, err := time.ParseDuration(*maxP95Str); err == nil {
				cfg.Thresholds.MaxP95 = p95Dur
			}
		}
		if *maxErrorRate >= 0 {
			cfg.Thresholds.MaxErrorRate = *maxErrorRate
		}
		if *breakToCrash {
			cfg.Pattern = types.PatternBreakToCrash
			cfg.Stress = types.StressConfig{
				Enabled:               true,
				InitialConcurrency:    5,
				StepConcurrency:       *stepVU,
				StepInterval:          2 * time.Second,
				CrashErrorRate:        75.0,
				CrashConsecutiveFails: 25,
				CrashMaxLatency:       10 * time.Second,
			}
		}
	} else {
		fmt.Println("Error: Either --config or --url must be provided.")
		fmt.Println("Run 'goloadtest help' for usage instructions.")
		os.Exit(1)
	}

	// Setup graceful interruption handling (Ctrl+C)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal. Gracefully stopping workers...")
		cancel()
	}()

	fmt.Printf("\n🚀 Starting GoLoadTest Suite: %s\n", cfg.Name)
	fmt.Printf("   Endpoints: %d | Duration: %v | Concurrency: %d | Target RPS: %.0f\n\n",
		len(cfg.Endpoints), cfg.Duration, cfg.Concurrency, cfg.TargetRPS)

	eng := engine.New(cfg)

	// Live terminal progress display
	eng.SetProgressListener(func(snap engine.ProgressSnapshot) {
		pct := (snap.Elapsed.Seconds() / snap.TotalDuration.Seconds()) * 100.0
		if pct > 100.0 {
			pct = 100.0
		}
		fmt.Printf("\r⏳ [%5.1f%%] %s / %s | Reqs: %d | RPS: %5.1f | Lat: %6v | Err: %4.1f%% | VUs: %d   ",
			pct,
			snap.Elapsed.Round(time.Second),
			snap.TotalDuration.Round(time.Second),
			snap.TotalRequests,
			snap.CurrentRPS,
			snap.CurrentAvgLat.Round(time.Millisecond),
			snap.ErrorRate,
			snap.ActiveWorkers,
		)
	})

	result, err := eng.Run(ctx)
	fmt.Print("\r" + strings.Repeat(" ", 90) + "\r") // Clear progress line

	if err != nil {
		fmt.Printf("Test execution failed: %v\n", err)
		os.Exit(1)
	}

	// Bottleneck & SLA analysis
	analyzer.Analyze(cfg, &result)

	// Print Terminal Summary
	printConsoleReport(result)

	// Save reports
	if cfg.OutputFormat != "" && cfg.OutputFormat != "none" {
		err := reporter.SaveResults(result, cfg.OutputFormat, cfg.OutputPath)
		if err == nil {
			fmt.Printf("📁 Reports saved to: %s\n", filepath.Clean(cfg.OutputPath))
		}
	}

	// Exit Code: 0 if passed, 1 if SLA failed or aborted
	if !result.Passed || result.Aborted {
		os.Exit(1)
	}
	os.Exit(0)
}

func printConsoleReport(res types.TestResult) {
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	statusText := "✅ PASSED"
	if res.Aborted {
		statusText = fmt.Sprintf("⚠️  ABORTED (%s)", res.AbortReason)
	} else if !res.Passed {
		statusText = "❌ FAILED (SLA Thresholds Breached)"
	}
	fmt.Printf("  TEST RESULT: %s\n", statusText)
	fmt.Println("═══════════════════════════════════════════════════════════════════════")

	fmt.Printf("  Total Requests:   %-12d | Throughput:      %.1f RPS\n", res.TotalRequests, res.ThroughputRPS)
	fmt.Printf("  Success:          %-12d | Failed:          %d (%.2f%%)\n", res.SuccessRequests, res.FailedRequests, res.ErrorRate)
	fmt.Printf("  Mean Latency:     %-12v | Median (P50):    %v\n", res.Latency.Mean.Round(time.Microsecond), res.Latency.Median.Round(time.Microsecond))
	fmt.Printf("  P95 Latency:      %-12v | P99 Latency:     %v\n", res.Latency.P95.Round(time.Microsecond), res.Latency.P99.Round(time.Microsecond))
	fmt.Printf("  Min Latency:      %-12v | Max Latency:     %v\n", res.Latency.Min.Round(time.Microsecond), res.Latency.Max.Round(time.Microsecond))
	if res.System.PeakCPUPercent > 0 {
		fmt.Printf("  Peak CPU:         %-11.1f%% | Peak Memory:     %.1f MB\n", res.System.PeakCPUPercent, res.System.PeakMemoryMB)
	}

	if res.BreakingPoint != nil && res.BreakingPoint.Detected {
		fmt.Println("───────────────────────────────────────────────────────────────────────")
		fmt.Printf("  ⚡ BREAKING POINT: Server collapsed at Concurrency %d (Safe: ~%d VUs)\n", res.BreakingPoint.CrashConcurrency, res.BreakingPoint.MaxStableConcurrency)
		fmt.Printf("     Reason: %s\n", res.BreakingPoint.CrashReason)
	}

	fmt.Println("───────────────────────────────────────────────────────────────────────")

	// Print SLA Thresholds
	if len(res.Thresholds) > 0 {
		fmt.Println("  SLA THRESHOLDS:")
		for _, th := range res.Thresholds {
			mark := "✓ PASS"
			if !th.Passed {
				mark = "✗ FAIL"
			}
			fmt.Printf("    [%s] %-20s Target: %-15s Actual: %s\n", mark, th.Metric, th.Target, th.Actual)
		}
		fmt.Println("───────────────────────────────────────────────────────────────────────")
	}

	// Print Bottleneck Insights
	if len(res.Bottlenecks) > 0 {
		fmt.Println("  BOTTLENECK INSIGHTS & RECOMMENDATIONS:")
		for _, b := range res.Bottlenecks {
			fmt.Printf("    [%s] %s\n", b.Severity, b.Title)
			fmt.Printf("         Evidence:   %s\n", b.Evidence)
			fmt.Printf("         Suggestion: %s\n", b.Suggestion)
		}
		fmt.Println("───────────────────────────────────────────────────────────────────────")
	}

	// Status Codes
	if len(res.StatusCodes) > 0 {
		fmt.Print("  Status Codes: ")
		for code, count := range res.StatusCodes {
			fmt.Printf("[%d: %d] ", code, count)
		}
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════════════════════════")
	}
}
