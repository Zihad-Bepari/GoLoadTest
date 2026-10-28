# Contributing to GoLoadTest (CONTRIBUTING.md)

Thank you for your interest in contributing to **GoLoadTest**! We welcome bug reports, documentation enhancements, feature proposals, and pull requests.

This document outlines the development workflow, coding standards, and testing procedures.

---

## 📑 Table of Contents

- [Code of Conduct](#-code-of-conduct)
- [Development Environment Setup](#-development-environment-setup)
- [Project Directory Structure](#-project-directory-structure)
- [Coding Conventions & Design Principles](#-coding-conventions--design-principles)
- [Testing & Benchmarking](#-testing--benchmarking)
  - [Running Unit & Integration Tests](#running-unit--integration-tests)
  - [Race Detector Testing](#race-detector-testing)
  - [Running Benchmarks](#running-benchmarks)
  - [Linting and Static Analysis](#linting-and-static-analysis)
- [Submitting Issues](#-submitting-issues)
- [Pull Request Guidelines](#-pull-request-guidelines)
- [Security Vulnerability Disclosures](#-security-vulnerability-disclosures)

---

## 🤝 Code of Conduct

We are committed to providing a welcoming, inclusive, and harassment-free experience for everyone. Please be respectful, constructive, and collaborative in all communications.

---

## 💻 Development Environment Setup

### Prerequisites
- **Go:** 1.21 or newer (1.23+ recommended).
- **Git:** 2.x+
- **Node.js (Optional):** v18+ if working on the Node.js / NestJS adapters.
- **PHP (Optional):** 8.0+ if working on the PHP / Laravel adapter.

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/Zihad-Bepari/GoLoadTest.git
cd GoLoadTest

# Verify dependencies (Go standard library only)
go mod tidy
go mod verify

# Build the CLI executable
go build -o goloadtest.exe ./cmd/goloadtest   # Windows
# or:
go build -o goloadtest ./cmd/goloadtest       # Linux / macOS
```

---

## 📁 Project Directory Structure

```text
GoLoadTest/
├── cmd/
│   └── goloadtest/          # CLI application entry point (main.go)
├── pkg/
│   ├── analyzer/            # SLA threshold verification & bottleneck heuristics
│   ├── engine/              # Goroutine scheduler, traffic generators & execution loops
│   ├── metrics/             # Percentile calculators (p50-p999) & time-series buckets
│   ├── monitor/             # Host CPU and memory sampling
│   ├── reporter/            # Standalone HTML and JSON report generators
│   ├── requester/           # HTTP connection pool client with TLS & auth handling
│   ├── server/              # Web Studio UI server and SSE streaming
│   └── types/               # Public domain types, configs, and result models
├── adapters/
│   ├── nodejs/              # Node.js Express wrapper adapter & route discovery
│   ├── nestjs/              # NestJS GoLoadTestModule & injectable service
│   └── php-laravel/         # PHP library & Laravel Artisan command provider
├── examples/                # Runnable sample projects (Gin, Express, NestJS, Laravel)
├── .github/workflows/       # CI test matrix, security audit, and cross-compilation
├── goloadtest.go            # Root Go package exports (goloadtest.Run)
├── goloadtest_test.go       # Root end-to-end integration test
└── goloadtest_bench_test.go # Engine performance benchmarks
```

---

## 📐 Coding Conventions & Design Principles

1. **Zero External Dependencies in Go Core:**
   The core Go engine strictly relies on the Go Standard Library. Do not introduce third-party Go dependencies to `go.mod` without prior approval and discussion via an issue.
2. **Standard Formatting:**
   Always format code using `gofmt` or `goimports` prior to committing:
   ```bash
   gofmt -s -w .
   ```
3. **Effective Error Handling:**
   Never ignore errors silently (`_ = err`). Return descriptive error messages wrapped with context where appropriate.
4. **Concurrency Safety:**
   Any metric collection or shared state across worker goroutines must use atomic primitives (`sync/atomic`) or read-write locks (`sync.RWMutex`). Never introduce race conditions.
5. **Preserve Documentation:**
   Preserve existing docstrings, exported function comments, and code explanations.

---

## 🧪 Testing & Benchmarking

All pull requests must pass the existing test suite and adhere to CI standards.

### Running Unit & Integration Tests

Run all unit tests across all packages:

```bash
go test -v ./...
```

### Race Detector Testing

Before pushing changes involving concurrency, verify that no race conditions exist:

```bash
go test -v -race -count=1 ./...
```

### Running Benchmarks

Ensure modifications do not introduce performance regressions:

```bash
go test -bench=BenchmarkLoadEngine -run=^$ -benchmem -v .
```

### Linting and Static Analysis

The project enforces Go standard vetting, `staticcheck`, and vulnerability scanning in CI:

```bash
# Vet standard checks
go vet ./...

# Staticcheck (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
staticcheck ./...

# Govulncheck (install: go install golang.org/x/vuln/cmd/govulncheck@latest)
govulncheck ./...
```

---

## 🐛 Submitting Issues

### Bug Reports
When opening a bug report, please include:
- A clear, descriptive title.
- Go version (`go version`) and Operating System.
- Minimal reproducible example or configuration file (`loadtest.json`).
- Expected vs. actual behavior, including console logs or error outputs.

### Feature Requests
When proposing a feature:
- Describe the problem you are solving and why existing solutions are insufficient.
- Outline the proposed API or CLI design.
- Consider whether the proposed addition aligns with the zero-dependency standard library design philosophy.

---

## 🚀 Pull Request Guidelines

1. **Create a Feature Branch:** Branch off `main` with a descriptive name (e.g., `git checkout -b feat/custom-headers-support` or `fix/p99-calculation`).
2. **Write Unit Tests:** Add test coverage for any new features or bug fixes.
3. **Verify CI Checks:** Ensure `go test -race ./...` and `go vet ./...` pass locally.
4. **Keep PRs Focused:** Avoid bundling unrelated refactorings or formatting changes into a single PR.
5. **Provide a Meaningful PR Description:** Summarize what was changed, why, and how it was verified.

---

## 🔒 Security Vulnerability Disclosures

If you discover a security vulnerability, please do not disclose it via a public GitHub issue. Instead, report it privately to the maintainers or via GitHub Security Advisories.
