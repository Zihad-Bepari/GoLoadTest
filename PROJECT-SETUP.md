# GoLoadTest — Setup & Integration Guide

A simple, step-by-step guide to installing and using **GoLoadTest** in **Go**, **Node.js**, **NestJS**, and **PHP / Laravel**.

---

## 📑 Table of Contents
1. [Installation Summary](#1-installation-summary)
2. [CLI & Pre-Compiled Binaries](#2-cli--pre-compiled-binaries)
3. [Go Integration](#3-go-integration)
4. [Node.js & Express Integration](#4-nodejs--express-integration)
5. [NestJS Integration](#5-nestjs-integration)
6. [PHP & Laravel Integration](#6-php--laravel-integration)
7. [Configuration Options](#7-configuration-options)
8. [Production Tips](#8-production-tips)

---

## 1. Installation Summary

| Platform | Command | Notes |
| :--- | :--- | :--- |
| **Go** | `go get github.com/Zihad-Bepari/GoLoadTest` | Go 1.21 or higher |
| **Node.js** | `npm install github:Zihad-Bepari/GoLoadTest` | Node 18 or higher |
| **NestJS** | `npm install github:Zihad-Bepari/GoLoadTest` | NestJS 9 or higher |
| **PHP / Laravel** | `composer require zihad-bepari/goloadtest:dev-main` | PHP 7.4+ or 8.x |

---

## 2. CLI & Pre-Compiled Binaries

Pre-compiled binaries are ready in the `bin/` directory:
- Linux: `bin/goloadtest-linux-amd64` (and `arm64`)
- macOS: `bin/goloadtest-darwin-amd64` (and `arm64`)
- Windows: `bin/goloadtest-windows-amd64.exe`

### Run from Terminal:
```bash
# Test a single endpoint
goloadtest run --url http://localhost:3000/api/users --duration 10s --concurrency 50

# Launch interactive web UI
goloadtest ui

# Create starter config file
goloadtest init
```

---

## 3. Go Integration

Add GoLoadTest to your Go project:
```bash
go get github.com/Zihad-Bepari/GoLoadTest
```

### Example:
```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/Zihad-Bepari/GoLoadTest"
    "github.com/Zihad-Bepari/GoLoadTest/pkg/types"
)

func main() {
    cfg := goloadtest.DefaultConfig()
    cfg.Name = "Users API Benchmark"
    cfg.Duration = 5 * time.Second
    cfg.Concurrency = 30
    cfg.Endpoints = []types.EndpointSpec{
        {
            Name:   "Get All Users",
            URL:    "http://localhost:8080/api/users",
            Method: "GET",
        },
    }

    result, err := goloadtest.Run(context.Background(), cfg)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Done! Total requests: %d, RPS: %.1f, P95: %v\n",
        result.TotalRequests, result.ThroughputRPS, result.Latency.P95)
}
```

---

## 4. Node.js & Express Integration

Install via npm:
```bash
npm install github:Zihad-Bepari/GoLoadTest
```

### 1-Click Interactive Dashboard in Express (`server.js`):
```javascript
const express = require('express');
const { loadtestMiddleware } = require('goloadtest');

const app = express();

app.get('/api/users', (req, res) => res.json([{ id: 1, name: 'Alice' }]));

// Mount the load test web page at /loadtest
app.use('/loadtest', loadtestMiddleware());

app.listen(3000, () => {
  console.log('Open http://localhost:3000/loadtest to start testing');
});
```

---

## 5. NestJS Integration

Install via npm:
```bash
npm install github:Zihad-Bepari/GoLoadTest
```

### Setup Module (`app.module.ts`):
```typescript
import { Module } from '@nestjs/common';
import { GoLoadTestModule } from 'goloadtest/nestjs';

@Module({
  imports: [GoLoadTestModule],
})
export class AppModule {}
```

### Use in Controller or E2E Test:
```typescript
import { Controller, Post } from '@nestjs/common';
import { GoLoadTestService } from 'goloadtest/nestjs';

@Controller('admin/loadtest')
export class LoadTestController {
  constructor(private readonly loadTestService: GoLoadTestService) {}

  @Post('start')
  async start() {
    return await this.loadTestService.run({
      name: 'NestJS Microservice Benchmark',
      url: 'http://localhost:3000/api/users',
      duration: '10s',
      concurrency: 50,
    });
  }
}
```

---

## 6. PHP & Laravel Integration

Install via Composer:
```bash
composer require zihad-bepari/goloadtest:dev-main
```

### In Laravel:
GoLoadTest registers an Artisan command automatically:
```bash
# Auto-discovers all your Laravel routes and tests them
php artisan test:load

# Test a specific URL
php artisan test:load --url=http://127.0.0.1:8000/api/users --concurrency=20 --duration=10s
```

### In Plain PHP:
```php
require 'vendor/autoload.php';

use GoLoadTest\GoLoadTest;

$result = GoLoadTest::run([
    'name' => 'API Capacity Test',
    'url' => 'http://127.0.0.1:8000/api/health',
    'duration' => '10s',
    'concurrency' => 20,
]);

echo "Total Requests: " . $result['totalRequests'] . "\n";
echo "Throughput: " . $result['throughputRps'] . " RPS\n";
```

---

## 7. Configuration Options

| Option | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `name` | string | `"API Load Test"` | Test suite title |
| `url` | string | `""` | Single endpoint URL |
| `endpoints` | array | `[]` | List of multiple endpoints to test |
| `duration` | string | `"10s"` | Duration (e.g. `10s`, `1m`) |
| `concurrency`| int | `10` | Number of simultaneous virtual users |
| `pattern` | string | `"constant"` | `constant`, `ramp`, `spike`, or `break-to-crash` |
| `maxP95` | string | `"300ms"` | SLA maximum acceptable 95th percentile latency |
| `maxErrorRate`| float | `2.0` | SLA maximum error percentage threshold |

---

## 8. Production Tips

- **cPanel & Shared Hosting:** If `exec()` is disabled in `php.ini`, GoLoadTest automatically uses its built-in PHP in-process fallback engine (`curl_multi`).
- **Do Not Test Production from Itself:** Run load tests from your local machine, staging server, or an external VPS to avoid stealing CPU from your real users.
- **Linux File Permissions:** Executable permissions (`chmod 755`) are automatically handled on start.
