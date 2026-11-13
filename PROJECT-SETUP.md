# GoLoadTest — Project Setup & Integration Guide

This guide provides end-to-end instructions for installing, configuring, integrating, and deploying **GoLoadTest** across supported programming languages (**Go**, **Node.js/Express**, **NestJS**, **PHP/Laravel**) and hosting environments (**Local**, **Docker**, **VPS**, **cPanel/Shared Hosting**).

---

## 📑 Table of Contents

- [Supported Platforms & Installation Summary](#supported-platforms--installation-summary)
- [1. Prerequisites & Binary Installation](#1-prerequisites--binary-installation)
  - [Pre-Compiled Binaries](#pre-compiled-binaries)
  - [Building from Source](#building-from-source)
- [2. CLI Usage Reference](#2-cli-usage-reference)
  - [Initialize Configuration](#initialize-configuration)
  - [Run Tests via CLI](#run-tests-via-cli)
  - [Launch Embedded Web Studio](#launch-embedded-web-studio)
- [3. Go Native Library Integration](#3-go-native-library-integration)
  - [Go Installation](#go-installation)
  - [Standard Usage Example](#standard-usage-example)
  - [Gin / Chi / Fiber Test Example](#gin--chi--fiber-test-example)
  - [Automated SLA Regression Gate (CI/CD)](#automated-sla-regression-gate-cicd)
- [4. Node.js & Express Integration](#4-nodejs--express-integration)
  - [Node.js Installation](#nodejs-installation)
  - [1-Click Interactive Dashboard Middleware (`app.js`)](#1-click-interactive-dashboard-middleware-appjs)
  - [Programmatic API Execution](#programmatic-api-execution)
- [5. NestJS Integration](#5-nestjs-integration)
  - [NestJS Installation](#nestjs-installation)
  - [Module Setup](#module-setup)
  - [Service Injection & Controller Usage](#service-injection--controller-usage)
- [6. PHP & Laravel Integration](#6-php--laravel-integration)
  - [PHP / Laravel Installation](#php--laravel-installation)
  - [Artisan CLI Command](#artisan-cli-command)
  - [Programmatic PHP Execution](#programmatic-php-execution)
- [7. Complete Configuration Reference](#7-complete-configuration-reference)
- [8. Production & Deployment Guidelines](#8-production--deployment-guidelines)
  - [Docker Setup](#docker-setup)
  - [Linux VPS & Nginx](#linux-vps--nginx)
  - [cPanel / CloudLinux / LiteSpeed / Phusion Passenger](#cpanel--cloudlinux--litespeed--phusion-passenger)
- [9. Troubleshooting & FAQs](#9-troubleshooting--faqs)

---

## Supported Platforms & Installation Summary

| Platform | Package Manager | Direct GitHub Install Command | Minimum Runtime |
| :--- | :--- | :--- | :--- |
| **Go** | `go get` | `go get github.com/Zihad-Bepari/GoLoadTest` | Go 1.21+ |
| **Node.js** | `npm` / `yarn` / `pnpm` | `npm install github:Zihad-Bepari/GoLoadTest` | Node.js 18+ |
| **NestJS** | `npm` / `yarn` / `pnpm` | `npm install github:Zihad-Bepari/GoLoadTest` | NestJS 9+ |
| **PHP** | `composer` | `composer require zihad-bepari/goloadtest:dev-main` | PHP 7.4+ / 8.x |
| **Laravel** | `composer` | `composer require zihad-bepari/goloadtest:dev-main` | Laravel 9, 10, 11 |

---

## 1. Prerequisites & Binary Installation

GoLoadTest requires:
- **Go:** 1.21 or higher (if building from source or using the native Go API).
- **OS Support:** Linux (x86_64, ARM64), macOS (Intel, Apple Silicon), Windows (x64).

### Pre-Compiled Binaries

Static binaries are located in the repository's `bin/` directory:

| Operating System | Architecture | Binary Path |
| :--- | :--- | :--- |
| **Linux** | AMD64 (x86_64) | `bin/goloadtest-linux-amd64` |
| **Linux** | ARM64 (aarch64) | `bin/goloadtest-linux-arm64` |
| **macOS** | Intel (AMD64) | `bin/goloadtest-darwin-amd64` |
| **macOS** | Apple Silicon (M1/M2/M3) | `bin/goloadtest-darwin-arm64` |
| **Windows** | AMD64 (x64) | `bin/goloadtest-windows-amd64.exe` |

On Linux and macOS, grant executable permissions before running:
```bash
chmod +x bin/goloadtest-linux-amd64
sudo mv bin/goloadtest-linux-amd64 /usr/local/bin/goloadtest
```

### Building from Source

To compile the latest standalone binary directly from source code:

```bash
git clone https://github.com/Zihad-Bepari/GoLoadTest.git
cd GoLoadTest

# Build optimized static binary
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/goloadtest ./cmd/goloadtest

# Verify installation
./bin/goloadtest version
```

---

## 2. CLI Usage Reference

### Initialize Configuration

Generate a starter `loadtest.json` configuration file in your current directory:
```bash
goloadtest init
```

### Run Tests via CLI

**Run with configuration file:**
```bash
goloadtest run --config loadtest.json
```

**Run quick ad-hoc test against a single URL:**
```bash
goloadtest run --url https://api.example.com/health --duration 30s --concurrency 50
```

**Run with SLA constraints and custom report output:**
```bash
goloadtest run \
  --config loadtest.json \
  --output ./test-results \
  --format all \
  --max-p95 250ms \
  --max-error 1.0
```

### Launch Embedded Web Studio

Start the zero-dependency, real-time web dashboard server:
```bash
goloadtest ui --port 8080
```
Open `http://localhost:8080` in your browser to configure endpoints, trigger live runs, and observe real-time throughput curves.

---

## 3. Go Native Library Integration

### Go Installation

Add the library module directly from GitHub into your `go.mod`:
```bash
go get github.com/Zihad-Bepari/GoLoadTest
```

### Standard Usage Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Zihad-Bepari/GoLoadTest"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func main() {
	cfg := goloadtest.DefaultConfig()
	cfg.Name = "Production API Stress Test"
	cfg.Duration = 30 * time.Second
	cfg.Concurrency = 100
	cfg.Pattern = types.PatternRamp
	cfg.RampUp = 5 * time.Second
	cfg.RampDown = 5 * time.Second

	cfg.Endpoints = []types.EndpointSpec{
		{
			Name:    "Public Products API",
			URL:     "https://api.example.com/v1/products",
			Method:  "GET",
			Timeout: 5 * time.Second,
		},
	}

	cfg.Thresholds = types.Thresholds{
		MaxP95:       200 * time.Millisecond,
		MaxErrorRate: 1.0, // max 1% error
	}

	cfg.OutputFormat = "json"
	cfg.OutputPath = "./results"

	result, err := goloadtest.Run(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Test run failed: %v", err)
	}

	fmt.Printf("Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("Throughput:     %.1f RPS\n", result.ThroughputRPS)
	fmt.Printf("P95 Latency:    %v\n", result.Latency.P95)
	fmt.Printf("SLA Passed:     %t\n", result.Passed)
}
```

### Gin / Chi / Fiber Test Example

To benchmark an in-memory HTTP server instance directly within a Go unit test:

```go
package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/Zihad-Bepari/GoLoadTest"
	"github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func setupRouter() *gin.Engine {
	r := gin.New()
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "pong"})
	})
	return r
}

func TestAPIPerformance(t *testing.T) {
	ts := httptest.NewServer(setupRouter())
	defer ts.Close()

	cfg := goloadtest.DefaultConfig()
	cfg.Duration = 5 * time.Second
	cfg.Concurrency = 25
	cfg.Endpoints = []types.EndpointSpec{
		{Name: "Ping", URL: ts.URL + "/api/ping", Method: "GET"},
	}

	res, err := goloadtest.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	if res.ErrorRate > 0 {
		t.Errorf("Expected 0 errors, got %.2f%%", res.ErrorRate)
	}
}
```

### Automated SLA Regression Gate (CI/CD)

```go
// Compare current build results against baseline to block regressions
baseline := loadPreviousResult("baseline.json")
current, _ := goloadtest.Run(context.Background(), cfg)

regression := goloadtest.CompareBaseline(current, baseline)
if regression.HasRegression {
    log.Fatalf("Build failed: Performance regressed by %.1f%%", regression.P95ChangePercent)
}
```

---

## 4. Node.js & Express Integration

### Node.js Installation

Install directly from GitHub into your Node.js project:

```bash
# Standard GitHub install:
npm install github:Zihad-Bepari/GoLoadTest
```

> **Note:** If you use yarn or pnpm:
> `yarn add github:Zihad-Bepari/GoLoadTest` or `pnpm add github:Zihad-Bepari/GoLoadTest`

### 1-Click Interactive Dashboard Middleware (`app.js`)

Mount the built-in middleware inside your Express application (`app.js` or `server.js`). It automatically introspects your router stack, detects declared routes, and renders an interactive dashboard:

```javascript
// app.js
const express = require('express');
const { loadtestMiddleware } = require('goloadtest');

const app = express();

// Your application routes
app.use('/api/v1/products', productsRouter);
app.use('/api/v1/users', usersRouter);

// 🚀 Mount GoLoadTest Interactive Studio at /loadtest
app.use('/loadtest', loadtestMiddleware({
  name: 'API Stress Test Studio',
  duration: '20m',               // Open-ended duration for break-to-crash
  stepConcurrency: 60,           // Controlled ramp rate (+60 VUs/sec)
  crashThreshold: 60.0           // Error rate triggering server breakdown
}));

app.listen(5000, () => {
  console.log('App running. Open http://localhost:5000/loadtest for load testing');
});
```

### Programmatic API Execution

You can also trigger stress tests programmatically from script files:

```javascript
const { runLoadTest, autoTest } = require('goloadtest');
const app = require('./app');

async function benchmark() {
  // Option A: Automatically discover and stress all routes in app
  const result = await autoTest(app, {
    duration: '30s',
    concurrency: 200,
    breakUntilCrash: true
  });

  console.log(`Throughput: ${result.throughputRps} RPS`);
  console.log(`Total Handled: ${result.totalRequests}`);
}

benchmark();
```

---

## 5. NestJS Integration

### NestJS Installation

Install directly from GitHub into your NestJS project:

```bash
npm install github:Zihad-Bepari/GoLoadTest
```

### Module Setup

Import `GoLoadTestModule` into your root `AppModule`:

```typescript
import { Module } from '@nestjs/common';
import { GoLoadTestModule } from 'goloadtest/adapters/nestjs/goloadtest.module';
import { ProductsModule } from './products/products.module';

@Module({
  imports: [
    ProductsModule,
    GoLoadTestModule,
  ],
})
export class AppModule {}
```

### Service Injection & Controller Usage

Inject `GoLoadTestService` into any testing controller or E2E test suite:

```typescript
import { Controller, Post, Get, INestApplication } from '@nestjs/common';
import { GoLoadTestService } from 'goloadtest/adapters/nestjs/goloadtest.service';

@Controller('admin/loadtest')
export class LoadTestController {
  constructor(private readonly loadTestService: GoLoadTestService) {}

  @Post('start')
  async runStressTest() {
    return await this.loadTestService.run({
      name: 'NestJS Microservice Capacity Test',
      endpoints: [
        { name: 'Products Catalog', url: 'http://localhost:3000/products', method: 'GET' },
      ],
      duration: '30s',
      concurrency: 150,
      breakUntilCrash: true,
    });
  }
}
```

---

## 6. PHP & Laravel Integration

### PHP / Laravel Installation

Install GoLoadTest directly from GitHub using Composer:

#### Option A: Quick Command Line (Recommended)
Run these two commands in your Laravel terminal:
```bash
composer config repositories.goloadtest vcs https://github.com/Zihad-Bepari/GoLoadTest.git
composer require zihad-bepari/goloadtest:dev-main
```

#### Option B: Add to `composer.json` Manually
Add GoLoadTest as a VCS repository in your `composer.json`:

```json
{
  "repositories": [
    {
      "type": "vcs",
      "url": "https://github.com/Zihad-Bepari/GoLoadTest.git"
    }
  ],
  "require": {
    "zihad-bepari/goloadtest": "dev-main"
  }
}
```

Then run:
```bash
composer update
```

### Artisan CLI Command

The package automatically registers the `test:load` Artisan command:

```bash
# Auto-discover all registered Laravel routes and run stress test
php artisan test:load
php artisan test:load

# Test specific endpoint with custom duration and concurrency
php artisan test:load --url=http://127.0.0.1:8000/api/users --duration=30s --concurrency=100

# Run continuous breaking point test until server drops sockets
php artisan test:load --break-to-crash --max-p95=200ms
```

### Programmatic PHP Execution

```php
<?php

use GoLoadTest\GoLoadTest;

$result = GoLoadTest::run([
    'name' => 'Laravel Payment API Stress Test',
    'endpoints' => [
        [
            'name' => 'Get Invoices',
            'url' => 'http://127.0.0.1:8000/api/v1/invoices',
            'method' => 'GET',
        ],
    ],
    'duration' => '20s',
    'concurrency' => 50,
    'breakUntilCrash' => true,
]);

echo "Throughput: " . $result['throughputRps'] . " RPS\n";
echo "Max Stable Concurrency: " . ($result['breakingPoint']['maxStableConcurrency'] ?? 'N/A') . "\n";
```

---

## 7. Complete Configuration Reference

Below is the complete reference for `loadtest.json` and the programmatic configuration options:

| Property | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `name` | string | `"API Load Test"` | Descriptive suite name |
| `baseUrl` | string | `""` | Optional base URL prepended to relative endpoint paths |
| `duration` | string | `"25s"` | Total run duration (e.g. `"30s"`, `"5m"`, `"20m"`) |
| `concurrency` | integer | `10` | Maximum virtual user workers (goroutines) |
| `targetRps` | float | `0` | Requests/sec rate limit (`0` = unthrottled) |
| `pattern` | string | `"constant"` | Load shape: `"constant"`, `"ramp"`, `"spike"`, `"step"`, `"break-to-crash"` |
| `rampUp` | string | `"0s"` | Duration to scale from 0 to full concurrency |
| `rampDown` | string | `"0s"` | Duration to scale down to 0 at end of test |
| `allowInsecureTLS`| boolean | `false` | Disable TLS certificate checks (Test environments only) |
| `outputFormat` | string | `"all"` | Output report formats: `"json"`, `"html"`, `"all"`, `"none"` |
| `outputPath` | string | `"./results"` | Destination directory for reports |
| **`endpoints`** | array | `[]` | List of target API definitions: |
| `endpoints[].name` | string | `"Endpoint"` | Endpoint label |
| `endpoints[].url` | string | *(Required)* | Full target URL |
| `endpoints[].method`| string | `"GET"` | HTTP Method (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`) |
| `endpoints[].headers`| object | `{}` | Key-value pairs of HTTP request headers |
| `endpoints[].body` | string | `""` | Raw string or JSON payload |
| `endpoints[].timeout`| string | `"10s"` | Per-request timeout duration |
| `endpoints[].weight` | integer | `1` | Relative traffic weighting in multi-route test |
| **`stress`** | object | `{}` | Settings for `break-to-crash` pattern: |
| `stress.initialConcurrency` | integer | `25` | Starting virtual user concurrency |
| `stress.stepConcurrency` | integer | `60` | Concurrency increment per time step |
| `stress.stepInterval` | string | `"1s"` | Frequency of concurrency step increases |
| `stress.crashConsecutiveFails` | integer | `30` | Dropped socket threshold considered server breakdown |
| `stress.crashErrorRate` | float | `60.0` | Error percentage threshold considered crash |
| **`thresholds`** | object | `{}` | SLA assertions: |
| `thresholds.maxP95` | string | `""` | Maximum allowable P95 response time (e.g. `"200ms"`) |
| `thresholds.maxErrorRate` | float | `-1` | Maximum allowable failure percentage (e.g. `1.0` for 1%) |

---

## 8. Production & Deployment Guidelines

### Docker Setup

A minimal multi-stage `Dockerfile` for executing containerized GoLoadTest runs:

```dockerfile
# Stage 1: Build static GoLoadTest binary
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o goloadtest ./cmd/goloadtest

# Stage 2: Runtime image
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/goloadtest /usr/local/bin/goloadtest
COPY loadtest.json ./

ENTRYPOINT ["goloadtest", "run", "--config", "loadtest.json"]
```

### Linux VPS & Nginx

When deploying GoLoadTest's embedded Web Studio on a Linux VPS behind Nginx:

```nginx
server {
    listen 80;
    server_name loadtest.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;

        # Required for Server-Sent Events (SSE) live updates
        proxy_set_header Connection '';
        proxy_set_header Cache-Control 'no-cache';
        proxy_buffering off;
        proxy_read_timeout 86400s;
    }
}
```

### cPanel / CloudLinux / LiteSpeed / Phusion Passenger

When running the Express middleware (`/loadtest`) inside a shared hosting cPanel environment:

1. **LiteSpeed Timeout Protection:** GoLoadTest's Node adapter automatically uses **Async Background Dispatch**. The initial `POST /api/start` responds in 5ms, and the browser polls `/api/status` every second. This completely prevents LiteSpeed's default 30-second gateway cutoff.
2. **OS Thread Caps (`ulimit -u`):** CloudLinux CageFS restricts the number of active OS threads per user account (often capped at 70). GoLoadTest's compiled binary includes:
   ```go
   debug.SetMaxThreads(45)
   runtime.GOMAXPROCS(2)
   ```
   This prevents `runtime: failed to create new OS thread (errno=11)` errors during high-concurrency scaling.
3. **Executable Permissions:** Ensure the pre-bundled binary in `src/libs/goloadtest/bin/goloadtest-linux-amd64` has executable permissions:
   ```bash
   chmod 755 src/libs/goloadtest/bin/goloadtest-linux-amd64
   ```

---

## 9. Troubleshooting & FAQs

### Q: Why do I see `Target Server Offline or Unreachable (Connection Refused)`?
**Cause:** The target backend server is not running, or the host firewall blocked outbound requests.
**Fix:**
- Verify your target backend is active (`curl http://localhost:5000/health`).
- On cPanel, ensure your server can make loopback requests to its own domain.

### Q: Why does my test stop with `Circuit Breaker Tripped: 30 consecutive failed requests`?
**Cause:** This is intended behavior in `break-to-crash` mode. The target application's connection backlog saturated, and the operating system rejected new TCP connections.
**Fix:** Look at the `Breaking Point Identified` summary card to find your maximum stable user capacity.

### Q: Why do I see `dial tcp: lookup ... i/o timeout`?
**Cause:** Local DNS resolver saturation under high concurrency.
**Fix:** Use IP addresses (e.g. `http://127.0.0.1:5000`) instead of hostnames for high-concurrency local tests.
