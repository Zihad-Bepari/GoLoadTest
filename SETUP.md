# GoLoadTest Installation & Integration Guide (SETUP.md)

This guide provides step-by-step instructions for installing and integrating **GoLoadTest** across all genuinely supported languages and frameworks.

---

## 📑 Table of Contents

- [Architectural Model: Native vs. Wrapper Adapters](#-architectural-model-native-vs-wrapper-adapters)
- [Prerequisite: Core Engine Binary](#-prerequisite-core-engine-binary)
- [1. Go (Native Integration)](#1-go-native-integration)
- [2. Node.js & Express (Process Wrapper)](#2-nodejs--express-process-wrapper)
- [3. NestJS (Process Wrapper Service)](#3-nestjs-process-wrapper-service)
- [4. PHP (Standalone Process Wrapper)](#4-php-standalone-process-wrapper)
- [5. Laravel (Artisan Command & Provider)](#5-laravel-artisan-command--provider)
- [Common Troubleshooting Across All Environments](#-common-troubleshooting-across-all-environments)

---

## 🏛 Architectural Model: Native vs. Wrapper Adapters

Before integrating, it is essential to understand how GoLoadTest interacts with different runtimes:

1. **Go:** The core engine is written natively in Go. Go applications import GoLoadTest directly as a standard Go library (`github.com/goloadtest/goloadtest`), requiring no external binary execution.
2. **Node.js, NestJS, PHP, and Laravel:** These integrations are **process-spawning wrapper adapters**. They construct typed test configurations, spawn the compiled `goloadtest` binary via child process execution (`child_process.spawn` or `exec`), and parse the generated JSON results.

> ⚠️ **Important:** For Node.js, NestJS, PHP, and Laravel, you **must** have the compiled `goloadtest` binary installed and accessible in your system `PATH` or in your project root.

---

## 📦 Prerequisite: Core Engine Binary

For non-Go integrations (or to use the CLI), obtain the `goloadtest` binary using one of the following methods:

### Option A: Build from Source (Recommended)
Requires Go 1.21 or later installed:

```bash
git clone https://github.com/Zihad-Bepari/GoLoadTest.git
cd GoLoadTest

# Build binary into your current directory or Go bin directory
go build -o goloadtest.exe ./cmd/goloadtest   # Windows
# or:
go build -o goloadtest ./cmd/goloadtest       # Linux / macOS
```

### Option B: Add to System PATH
Move the built binary to a folder in your system `$PATH` (e.g., `/usr/local/bin` on Linux/macOS, or a folder in your `Path` environment variable on Windows) so any adapter can locate it automatically:

```bash
# Verify the binary is accessible
goloadtest version
# Output: GoLoadTest v1.0.0 - Cross-Language API Performance Suite
```

---

## 1. Go (Native Integration)

### Prerequisites & Versions
- **Go Version:** 1.21, 1.22, 1.23, or 1.24.
- **Operating Systems:** Linux, macOS, Windows (x86_64 and ARM64).

### Installation
Add the module to your `go.mod` file:

```bash
go get github.com/goloadtest/goloadtest
```

### Minimal Working Example

Create a file named `loadtest_test.go`:

```go
package main_test

import (
	"context"
	"testing"
	"time"

	"github.com/goloadtest/goloadtest"
	"github.com/goloadtest/goloadtest/pkg/types"
)

func TestAPI_PerformanceSLA(t *testing.T) {
	// Initialize default configuration
	cfg := goloadtest.DefaultConfig()
	cfg.Name = "User Microservice SLA Verification"
	cfg.Duration = 5 * time.Second
	cfg.Concurrency = 20
	cfg.Pattern = types.PatternRamp
	cfg.RampUp = 1 * time.Second

	// Configure target endpoint
	cfg.Endpoints = []types.EndpointSpec{
		{
			Name:           "Get Users",
			URL:            "http://127.0.0.1:8080/api/users",
			Method:         "GET",
			Timeout:        2 * time.Second,
			ExpectedStatus: []int{200},
			Weight:         1,
		},
	}

	// Enforce strict SLA thresholds
	cfg.Thresholds = types.Thresholds{
		MaxP95:        50 * time.Millisecond,
		MaxErrorRate:  0.5, // max 0.5% errors
		MinThroughput: 100.0, // minimum 100 RPS
	}

	// Suppress output files during unit test execution
	cfg.OutputFormat = "none"

	// Run test
	result, err := goloadtest.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Load test execution failed: %v", err)
	}

	// Verify SLA compliance
	if !result.Passed {
		t.Errorf("Performance SLA breached! P95: %v, Error Rate: %.2f%%",
			result.Latency.P95, result.ErrorRate)
	}
}
```

### Configuration Options & Defaults

| Option | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `Name` | `string` | `"API Load Test"` | Test suite identifier |
| `Duration` | `time.Duration` | `10s` | Test duration |
| `Concurrency` | `int` | `10` | Concurrent worker goroutines (Virtual Users) |
| `Pattern` | `LoadPattern` | `"constant"` | `"constant"`, `"ramp"`, `"spike"`, `"step"`, `"break-to-crash"` |
| `TargetRPS` | `float64` | `0` | Rate limiter (0 = unlimited) |
| `OutputFormat` | `string` | `"all"` | `"all"`, `"json"`, `"html"`, `"none"` |
| `OutputPath` | `string` | `"./results"` | Destination directory for test reports |
| `AllowInsecureTLS` | `bool` | `false` | Disable TLS certificate checks for self-signed certificates |

---

## 2. Node.js & Express (Process Wrapper)

### Prerequisites & Versions
- **Node.js:** v18.0.0 or later (LTS recommended).
- **Core Binary:** `goloadtest` binary available in your project root or `$PATH`.

### Installation
From the repository root or your application directory:

```bash
# If using the local monorepo adapter
npm install ./adapters/nodejs
# Or if installed from local/private registry
npm install @goloadtest/core
```

Ensure the compiled `goloadtest` binary is either placed in your project root or added to `$PATH`.

### Minimal Working Example (Route Auto-Discovery)

Create a test script `test-express.js`:

```javascript
const express = require('express');
const { autoTest, extractRoutes } = require('@goloadtest/core');

const app = express();

app.get('/api/health', (req, res) => res.json({ status: 'ok' }));
app.get('/api/users', (req, res) => res.json([{ id: 1, name: 'Alice' }]));
app.post('/api/users', express.json(), (req, res) => res.status(201).json({ id: 2 }));

const server = app.listen(3000, async () => {
  console.log('Server started on http://127.0.0.1:3000');

  try {
    // Automatically extracts all routes and executes a stress test
    const result = await autoTest(app, {
      baseUrl: 'http://127.0.0.1:3000',
      duration: '5s',
      concurrency: 25,
      thresholds: {
        maxP95: '60ms',
        maxErrorRate: 1.0,
      },
    });

    console.log('Test Complete:');
    console.log(`- Total Requests: ${result.totalRequests}`);
    console.log(`- Throughput: ${result.throughputRps.toFixed(1)} RPS`);
    console.log(`- P95 Latency: ${result.latency.p95}ms`);
    console.log(`- Passed SLA: ${result.passed}`);

    if (!result.passed) {
      process.exitCode = 1;
    }
  } catch (err) {
    console.error('Load test failed:', err.message);
    process.exitCode = 1;
  } finally {
    server.close();
  }
});
```

### Express Middleware for Interactive Testing
GoLoadTest also includes an Express middleware to expose a test runner directly from your application:

```javascript
const express = require('express');
const { loadtestMiddleware } = require('@goloadtest/core');

const app = express();
app.use(loadtestMiddleware(app)); // mounts /__loadtest runner endpoint
```

---

## 3. NestJS (Process Wrapper Service)

### Prerequisites & Versions
- **NestJS:** v9.x or v10.x.
- **Node.js:** v18.0.0 or later.
- **Core Binary:** `goloadtest` binary available in your project root or `$PATH`.

### Installation
Import the adapter from `adapters/nestjs` or include it in your NestJS project:

```bash
npm install ./adapters/nodejs
```

### Minimal Working Example (E2E Test)

Create `test/users.load.e2e-spec.ts`:

```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { INestApplication } from '@nestjs/common';
import { AppModule } from '../src/app.module';
import { GoLoadTestService } from '../adapters/nestjs/goloadtest.service';

describe('Users API (Performance / Load Test)', () => {
  let app: INestApplication;
  const loadTester = new GoLoadTestService();

  beforeAll(async () => {
    const moduleFixture: TestingModule = await Test.createTestingModule({
      imports: [AppModule],
    }).compile();

    app = moduleFixture.createNestApplication();
    await app.listen(0); // dynamically bind to an ephemeral port
  });

  afterAll(async () => {
    await app.close();
  });

  it('should pass SLA thresholds across all discovered routes', async () => {
    // autoTest automatically queries app.getUrl() and extracts NestJS routes
    const result = await loadTester.autoTest(app, {
      duration: '5s',
      concurrency: 30,
      thresholds: {
        maxP95: '80ms',
        maxErrorRate: 1.0,
      },
    });

    expect(result.passed).toBe(true);
    expect(result.errorRate).toBeLessThan(1.0);
  }, 30000);
});
```

---

## 4. PHP (Standalone Process Wrapper)

### Prerequisites & Versions
- **PHP Version:** 7.4, 8.0, 8.1, 8.2, or 8.3.
- **Function Availability:** `exec` or `shell_exec` enabled in `php.ini` (not blocked in `disable_functions`).
- **Core Binary:** `goloadtest` binary in system `$PATH` or project root.

### Installation
In your `composer.json`:

```json
{
  "repositories": [
    {
      "type": "path",
      "url": "path/to/GoLoadTest/adapters/php-laravel"
    }
  ],
  "require": {
    "goloadtest/goloadtest-php": "*"
  }
}
```

Or directly require `GoLoadTest.php`:

```php
require_once __DIR__ . '/adapters/php-laravel/src/GoLoadTest.php';
```

### Minimal Working Example

```php
<?php

require_once __DIR__ . '/vendor/autoload.php';

use GoLoadTest\GoLoadTest;

try {
    $result = GoLoadTest::run([
        'name' => 'PHP Microservice Performance Benchmark',
        'url' => 'http://127.0.0.1:8000/api/users',
        'method' => 'GET',
        'duration' => '5s',
        'concurrency' => 20,
        'thresholds' => [
            'maxP95' => '100ms',
            'maxErrorRate' => 1.0,
        ],
    ]);

    echo "Requests: " . $result['totalRequests'] . "\n";
    echo "Throughput: " . round($result['throughputRps'], 1) . " RPS\n";
    echo "Passed SLA: " . ($result['passed'] ? 'YES' : 'NO') . "\n";

    if (!$result['passed']) {
        exit(1);
    }
} catch (\Exception $e) {
    fwrite(STDERR, "Execution Error: " . $e->getMessage() . "\n");
    exit(1);
}
```

---

## 5. Laravel (Artisan Command & Provider)

### Prerequisites & Versions
- **Laravel:** 9.x, 10.x, or 11.x.
- **PHP:** 8.0 or higher.
- **Core Binary:** `goloadtest` binary in project root or system `$PATH`.

### Setup
The package auto-registers via Laravel Package Discovery. If manual registration is required, add the service provider to `config/app.php`:

```php
'providers' => [
    // ...
    GoLoadTest\GoLoadTestServiceProvider::class,
],
```

### Running via Artisan Command

Run an ad-hoc load test against a specific endpoint:
```bash
php artisan test:load --url=http://127.0.0.1:8000/api/users --duration=10s --concurrency=30 --max-p95=100ms
```

Run in **Break-Until-Crash** mode with automatic route discovery (omitting `--url` will inspect all declared Laravel routes):
```bash
php artisan test:load --duration=20s --break-to-crash
```

### Artisan Command Options Reference

| Option | Default | Description |
| :--- | :--- | :--- |
| `--url` | *empty* (auto-discovers routes) | Target endpoint URL |
| `--duration` | `20s` | Test duration |
| `--concurrency` | `50` | Concurrent virtual users |
| `--break-to-crash`| `false` | Dynamically steps up traffic until server breakdown |
| `--rps` | `0` (unthrottled) | Rate limit in requests per second |
| `--max-p95` | `300ms` | SLA P95 latency threshold |
| `--max-error` | `2.0` | SLA maximum error rate percentage |

---

## 🛠 Common Troubleshooting Across All Environments

### 1. `Failed to execute GoLoadTest binary (goloadtest): ENOENT` (Node.js / NestJS)
- **Cause:** The Node.js child process cannot find the `goloadtest` executable.
- **Resolution:** Compile the binary with `go build -o goloadtest ./cmd/goloadtest` and place it in your project's working directory, or move it into a directory listed in your system `$PATH` (e.g., `/usr/local/bin`).

### 2. `exec() has been disabled for security reasons` (PHP / Laravel)
- **Cause:** The PHP environment has disabled process execution functions.
- **Resolution:** Edit your `php.ini` file, find `disable_functions`, and remove `exec` and `shell_exec`.

### 3. `bind: address already in use` when starting Web Studio UI
- **Cause:** Another process is already listening on default port `8989`.
- **Resolution:** Specify an alternative port using the `--port` flag:
  ```bash
  goloadtest ui --port 9090
  ```

### 4. `x509: certificate signed by unknown authority`
- **Cause:** The target API uses a self-signed or local development TLS certificate.
- **Resolution:** In your configuration, set `"allowInsecureTls": true` (or `--insecure` in CLI). Only do this in isolated local development environments.
