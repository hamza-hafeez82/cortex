# Contributing to Cortex

Thank you for your interest in contributing. This document explains how to get set up, how to add new detectors, and how to submit quality PRs.

## Getting started

**Prerequisites:** Go 1.22+, Make, golangci-lint

```bash
git clone https://github.com/hamza-hafeez82/cortex.git
cd cortex
go mod tidy
make build
./dist/cortex --version
```

## Project structure

```
cmd/cortex/          — CLI entry point
internal/
  engine/recon/      — Stage 1: reconnaissance
  engine/security/   — Stage 2: security scanning
  engine/architecture/ — Stage 3: architecture analysis
  ai/                — AI provider abstraction
  walker/            — concurrent file walker
  parser/            — language-aware parsers
  report/            — issue codes, formatting, output
  tui/               — Bubbletea terminal UI
pkg/detector/        — public Detector interface
docs/detectors/      — one .md doc per detector category
testdata/            — fixture repos for testing
```

## Adding a new detector

### 1. Assign an issue code

Check `docs/issue-codes.md` and pick the next available code in the appropriate category (`CX-SEC`, `CX-ARCH`, `CX-DEP`).

### 2. Implement the interface

Create a new file in `pkg/detector/detectors/`:

```go
package detectors

import "github.com/hamza-hafeez82/cortex/pkg/detector"

type HardcodedSecretsDetector struct{}

func (d *HardcodedSecretsDetector) ID() string            { return "CX-SEC-001" }
func (d *HardcodedSecretsDetector) Name() string          { return "Hardcoded Secrets" }
func (d *HardcodedSecretsDetector) Category() string      { return "security" }
func (d *HardcodedSecretsDetector) Severity() string      { return "critical" }
func (d *HardcodedSecretsDetector) Run(ctx *detector.ScanContext) []detector.Issue {
    // implementation
}
```

### 3. Add fixture files

Add vulnerable and clean fixture files to `testdata/` so your detector can be tested against real examples:

```
testdata/
  fixtures/
    hardcoded-secrets/
      vulnerable.go   ← should trigger the detector
      clean.go        ← should not trigger the detector
```

### 4. Write tests

```go
func TestHardcodedSecretsDetector(t *testing.T) {
    // load fixture, run detector, assert issues
}
```

### 5. Document it

Add a row to `docs/issue-codes.md` and create `docs/detectors/cx-sec-001.md`.

## Commit format

We follow Conventional Commits:

```
type(scope): short description

optional body
```

Types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`

Examples:
```
feat(security): add CX-SEC-001 hardcoded secrets detector
fix(walker): skip symlinks that cause infinite loops
docs(readme): add Ollama setup instructions
```

## Running tests

```bash
make test          # all tests with race detector
make test-cover    # tests + HTML coverage report
make lint          # golangci-lint
make vet           # go vet
```

## Submitting a PR

1. Fork the repo and create a branch: `git checkout -b feat/cx-sec-009-xxe`
2. Make your changes and write tests
3. Run `make test` and `make lint` — both must pass
4. Open a PR against `main` using the PR template
5. Wait for CI to go green

All PRs require one review before merging. For large changes, open an issue first to discuss the approach.