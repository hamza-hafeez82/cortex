package recon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hamza-hafeez82/cortex/internal/engine/recon"
	"github.com/hamza-hafeez82/cortex/internal/walker"
)

// makeRepo creates a temp directory from a map of path → content,
// walks it, and returns the RepoMap.
func makeRepo(t *testing.T, files map[string]string) *walker.RepoMap {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	w := walker.New(walker.DefaultOptions())
	repo, err := w.Walk(root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	return repo
}

// ── Dependency tests ──────────────────────────────────────────────────────────

func TestParsePackageJSON(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"package.json": `{
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "^4.17.21"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}`,
	})

	deps := recon.ParseDependencies(repo)
	if len(deps) != 3 {
		t.Errorf("expected 3 deps, got %d", len(deps))
	}

	byName := make(map[string]recon.Dependency)
	for _, d := range deps {
		byName[d.Name] = d
	}

	if byName["express"].Ecosystem != "npm" {
		t.Error("expected express ecosystem to be npm")
	}
	if byName["express"].Dev {
		t.Error("express should not be a dev dependency")
	}
	if !byName["jest"].Dev {
		t.Error("jest should be a dev dependency")
	}
}

func TestParseGoMod(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"go.mod": `module github.com/example/app

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/redis/go-redis/v9 v9.0.0
)

require github.com/stretchr/testify v1.8.4
`,
	})

	deps := recon.ParseDependencies(repo)
	if len(deps) != 3 {
		t.Errorf("expected 3 deps, got %d", len(deps))
	}

	byName := make(map[string]recon.Dependency)
	for _, d := range deps {
		byName[d.Name] = d
	}

	if byName["github.com/gin-gonic/gin"].Version != "v1.9.1" {
		t.Errorf("expected gin version v1.9.1, got %q", byName["github.com/gin-gonic/gin"].Version)
	}
	if byName["github.com/gin-gonic/gin"].Ecosystem != "go" {
		t.Error("expected go ecosystem")
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"requirements.txt": `# Production deps
fastapi==0.100.0
uvicorn>=0.22.0
sqlalchemy~=2.0.0

# skip options
--index-url https://pypi.org/simple
-r base.txt
`,
	})

	deps := recon.ParseDependencies(repo)
	if len(deps) != 3 {
		t.Errorf("expected 3 deps, got %d", len(deps))
	}

	byName := make(map[string]recon.Dependency)
	for _, d := range deps {
		byName[d.Name] = d
	}

	if byName["fastapi"].Version != "==0.100.0" {
		t.Errorf("expected fastapi version ==0.100.0, got %q", byName["fastapi"].Version)
	}
	if byName["fastapi"].Ecosystem != "pypi" {
		t.Error("expected pypi ecosystem")
	}
}

func TestParseCargoToml(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"Cargo.toml": `[package]
name = "my-app"
version = "0.1.0"

[dependencies]
serde = { version = "1.0", features = ["derive"] }
tokio = "1.28"

[dev-dependencies]
mockall = "0.11"
`,
	})

	deps := recon.ParseDependencies(repo)

	byName := make(map[string]recon.Dependency)
	for _, d := range deps {
		byName[d.Name] = d
	}

	if _, ok := byName["tokio"]; !ok {
		t.Error("expected tokio in deps")
	}
	if byName["tokio"].Ecosystem != "cargo" {
		t.Error("expected cargo ecosystem")
	}
	if !byName["mockall"].Dev {
		t.Error("mockall should be a dev dependency")
	}
}

// ── Tech stack tests ──────────────────────────────────────────────────────────

func TestDetectTechStackFromFiles(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"package.json":       `{"name":"app"}`,
		"tsconfig.json":      `{}`,
		"next.config.js":     `module.exports = {}`,
		"tailwind.config.js": `module.exports = {}`,
	})

	techs := recon.DetectTechStack(repo)
	techNames := make(map[string]bool)
	for _, t2 := range techs {
		techNames[t2.Name] = true
	}

	for _, expected := range []string{"Node.js", "TypeScript", "Next.js", "Tailwind CSS"} {
		if !techNames[expected] {
			t.Errorf("expected %q in tech stack", expected)
		}
	}
}

func TestDetectTechStackFromImports(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"src/app.ts": `import express from "express"
import { PrismaClient } from "@prisma/client"

const app = express()
`,
	})

	techs := recon.DetectTechStack(repo)
	techNames := make(map[string]bool)
	for _, t2 := range techs {
		techNames[t2.Name] = true
	}

	if !techNames["Express"] {
		t.Error("expected Express in tech stack from imports")
	}
	if !techNames["Prisma"] {
		t.Error("expected Prisma in tech stack from imports")
	}
}

func TestDetectInfrastructure(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"Dockerfile":               "FROM node:18\n",
		"docker-compose.yml":       "version: '3'\n",
		".github/workflows/ci.yml": "name: CI\n",
		"infra/main.tf":            `provider "aws" {}`,
	})

	runner := recon.NewRunner()
	result, err := runner.Run(repo)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !result.HasDocker {
		t.Error("expected HasDocker = true")
	}
	if !result.HasDockerCompose {
		t.Error("expected HasDockerCompose = true")
	}
	if !result.HasCI {
		t.Error("expected HasCI = true")
	}
	if !result.HasTerraform {
		t.Error("expected HasTerraform = true")
	}

	ciFound := false
	for _, ci := range result.CIProviders {
		if ci == "github-actions" {
			ciFound = true
		}
	}
	if !ciFound {
		t.Error("expected github-actions in CI providers")
	}
}

// ── Endpoint tests ────────────────────────────────────────────────────────────

func TestDetectExpressEndpoints(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"src/routes.js": `const express = require('express')
const router = express.Router()

router.get('/users', getUsers)
router.post('/users', createUser)
router.put('/users/:id', updateUser)
router.delete('/users/:id', deleteUser)

module.exports = router
`,
	})

	endpoints := recon.DetectEndpoints(repo)
	if len(endpoints) != 4 {
		t.Errorf("expected 4 endpoints, got %d", len(endpoints))
	}

	methods := make(map[string]bool)
	for _, e := range endpoints {
		methods[e.Method] = true
	}
	for _, m := range []string{"GET", "POST", "PUT", "DELETE"} {
		if !methods[m] {
			t.Errorf("expected method %s", m)
		}
	}
}

func TestDetectFastAPIEndpoints(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"app/main.py": `from fastapi import FastAPI
app = FastAPI()

@app.get("/health")
async def health_check():
    return {"status": "ok"}

@app.post("/users")
async def create_user(user: UserCreate):
    pass
`,
	})

	endpoints := recon.DetectEndpoints(repo)
	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
	}
}

// ── Runner integration test ───────────────────────────────────────────────────

func TestRunnerFullScan(t *testing.T) {
	repo := makeRepo(t, map[string]string{
		"main.go": `package main
import "github.com/gin-gonic/gin"
func main() {
    r := gin.Default()
    r.GET("/ping", pingHandler)
}`,
		"go.mod": `module example.com/app
go 1.22
require github.com/gin-gonic/gin v1.9.1`,
		"Dockerfile": "FROM golang:1.22\n",
	})

	runner := recon.NewRunner()
	result, err := runner.Run(repo)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.TotalFiles == 0 {
		t.Error("expected TotalFiles > 0")
	}
	if len(result.Dependencies) == 0 {
		t.Error("expected at least one dependency from go.mod")
	}
	if !result.HasDocker {
		t.Error("expected HasDocker = true")
	}
	if result.Summary() == "" {
		t.Error("Summary() should not be empty")
	}
}

func TestRunnerNilRepo(t *testing.T) {
	runner := recon.NewRunner()
	_, err := runner.Run(nil)
	if err == nil {
		t.Error("expected error for nil repo")
	}
}
