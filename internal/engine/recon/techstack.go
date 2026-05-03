package recon

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/walker"
)

// techSignal maps a filename or path substring to a Technology.
type techSignal struct {
	signal     string // substring to look for in file path
	tech       Technology
	exactMatch bool // if true, match only the filename, not a path substring
}

// fileTechSignals are detected by the presence of specific files in the repo.
var fileTechSignals = []techSignal{
	// JavaScript / Node runtimes and tools
	{signal: "package.json", exactMatch: true, tech: Technology{Name: "Node.js", Category: "runtime", Language: "JavaScript", Confidence: "high"}},
	{signal: "next.config", tech: Technology{Name: "Next.js", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{signal: "nuxt.config", tech: Technology{Name: "Nuxt.js", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{signal: "vite.config", tech: Technology{Name: "Vite", Category: "tool", Language: "JavaScript", Confidence: "high"}},
	{signal: "webpack.config", tech: Technology{Name: "Webpack", Category: "tool", Language: "JavaScript", Confidence: "high"}},
	{signal: "tailwind.config", tech: Technology{Name: "Tailwind CSS", Category: "framework", Language: "CSS", Confidence: "high"}},
	{signal: "jest.config", tech: Technology{Name: "Jest", Category: "tool", Language: "JavaScript", Confidence: "high"}},
	{signal: "vitest.config", tech: Technology{Name: "Vitest", Category: "tool", Language: "JavaScript", Confidence: "high"}},
	{signal: ".eslintrc", tech: Technology{Name: "ESLint", Category: "tool", Language: "JavaScript", Confidence: "high"}},
	{signal: "tsconfig.json", exactMatch: true, tech: Technology{Name: "TypeScript", Category: "tool", Language: "TypeScript", Confidence: "high"}},

	// Python
	{signal: "requirements.txt", exactMatch: true, tech: Technology{Name: "pip", Category: "tool", Language: "Python", Confidence: "high"}},
	{signal: "pyproject.toml", exactMatch: true, tech: Technology{Name: "Python project", Category: "runtime", Language: "Python", Confidence: "high"}},
	{signal: "setup.py", exactMatch: true, tech: Technology{Name: "setuptools", Category: "tool", Language: "Python", Confidence: "high"}},
	{signal: "manage.py", exactMatch: true, tech: Technology{Name: "Django", Category: "framework", Language: "Python", Confidence: "high"}},
	{signal: "alembic.ini", exactMatch: true, tech: Technology{Name: "Alembic", Category: "tool", Language: "Python", Confidence: "high"}},

	// Go
	{signal: "go.mod", exactMatch: true, tech: Technology{Name: "Go module", Category: "runtime", Language: "Go", Confidence: "high"}},

	// Rust
	{signal: "Cargo.toml", exactMatch: true, tech: Technology{Name: "Rust/Cargo", Category: "runtime", Language: "Rust", Confidence: "high"}},

	// Java / JVM
	{signal: "pom.xml", exactMatch: true, tech: Technology{Name: "Maven", Category: "tool", Language: "Java", Confidence: "high"}},
	{signal: "build.gradle", tech: Technology{Name: "Gradle", Category: "tool", Language: "Java", Confidence: "high"}},

	// Infrastructure
	{signal: "Dockerfile", tech: Technology{Name: "Docker", Category: "tool", Language: "", Confidence: "high"}},
	{signal: "docker-compose", tech: Technology{Name: "Docker Compose", Category: "tool", Language: "", Confidence: "high"}},
	{signal: "kubernetes", tech: Technology{Name: "Kubernetes", Category: "tool", Language: "", Confidence: "medium"}},
	{signal: "k8s", tech: Technology{Name: "Kubernetes", Category: "tool", Language: "", Confidence: "medium"}},
	{signal: ".terraform", tech: Technology{Name: "Terraform", Category: "tool", Language: "", Confidence: "high"}},
	{signal: "main.tf", exactMatch: true, tech: Technology{Name: "Terraform", Category: "tool", Language: "", Confidence: "high"}},

	// CI providers
	{signal: ".github/workflows", tech: Technology{Name: "GitHub Actions", Category: "tool", Language: "", Confidence: "high"}},
	{signal: ".gitlab-ci", tech: Technology{Name: "GitLab CI", Category: "tool", Language: "", Confidence: "high"}},
	{signal: ".circleci", tech: Technology{Name: "CircleCI", Category: "tool", Language: "", Confidence: "high"}},
	{signal: "Jenkinsfile", exactMatch: true, tech: Technology{Name: "Jenkins", Category: "tool", Language: "", Confidence: "high"}},
	{signal: ".travis.yml", exactMatch: true, tech: Technology{Name: "Travis CI", Category: "tool", Language: "", Confidence: "high"}},
}

// importFrameworkSignals maps import substrings to framework Technologies.
// These require scanning file contents, not just filenames.
var importFrameworkSignals = []struct {
	substr string
	tech   Technology
}{
	// Go frameworks
	{"github.com/gin-gonic/gin", Technology{Name: "Gin", Category: "framework", Language: "Go", Confidence: "high"}},
	{"github.com/labstack/echo", Technology{Name: "Echo", Category: "framework", Language: "Go", Confidence: "high"}},
	{"github.com/gofiber/fiber", Technology{Name: "Fiber", Category: "framework", Language: "Go", Confidence: "high"}},
	{"github.com/go-chi/chi", Technology{Name: "Chi", Category: "framework", Language: "Go", Confidence: "high"}},
	{"go.mongodb.org/mongo-driver", Technology{Name: "MongoDB", Category: "database", Language: "Go", Confidence: "high"}},
	{"gorm.io/gorm", Technology{Name: "GORM", Category: "database", Language: "Go", Confidence: "high"}},
	{"github.com/redis/go-redis", Technology{Name: "Redis", Category: "database", Language: "Go", Confidence: "high"}},

	// Python frameworks
	{"from django", Technology{Name: "Django", Category: "framework", Language: "Python", Confidence: "high"}},
	{"import django", Technology{Name: "Django", Category: "framework", Language: "Python", Confidence: "high"}},
	{"from flask", Technology{Name: "Flask", Category: "framework", Language: "Python", Confidence: "high"}},
	{"import flask", Technology{Name: "Flask", Category: "framework", Language: "Python", Confidence: "high"}},
	{"from fastapi", Technology{Name: "FastAPI", Category: "framework", Language: "Python", Confidence: "high"}},
	{"import fastapi", Technology{Name: "FastAPI", Category: "framework", Language: "Python", Confidence: "high"}},
	{"import sqlalchemy", Technology{Name: "SQLAlchemy", Category: "database", Language: "Python", Confidence: "high"}},
	{"from sqlalchemy", Technology{Name: "SQLAlchemy", Category: "database", Language: "Python", Confidence: "high"}},

	// JavaScript/TypeScript frameworks (from import statements)
	{"from 'react'", Technology{Name: "React", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{`from "react"`, Technology{Name: "React", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{"from 'vue'", Technology{Name: "Vue.js", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{`from "vue"`, Technology{Name: "Vue.js", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{"from '@angular/core'", Technology{Name: "Angular", Category: "framework", Language: "TypeScript", Confidence: "high"}},
	{"from 'express'", Technology{Name: "Express", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{`from "express"`, Technology{Name: "Express", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{"require('express')", Technology{Name: "Express", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{`require("express")`, Technology{Name: "Express", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{"from 'next'", Technology{Name: "Next.js", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{"from 'svelte'", Technology{Name: "Svelte", Category: "framework", Language: "JavaScript", Confidence: "high"}},
	{"@prisma/client", Technology{Name: "Prisma", Category: "database", Language: "JavaScript", Confidence: "high"}},
	{"mongoose", Technology{Name: "Mongoose", Category: "database", Language: "JavaScript", Confidence: "medium"}},
}

// DetectTechStack analyzes a RepoMap and returns all detected technologies.
func DetectTechStack(repo *walker.RepoMap) []Technology {
	seen := make(map[string]bool)
	var techs []Technology

	add := func(t Technology, source string) {
		if seen[t.Name] {
			return
		}
		seen[t.Name] = true
		t.Source = source
		techs = append(techs, t)
	}

	// File-based detection
	for _, f := range repo.Files {
		for _, sig := range fileTechSignals {
			if sig.exactMatch {
				if f.Name == sig.signal {
					add(sig.tech, f.Path)
				}
			} else {
				if strings.Contains(f.Path, sig.signal) {
					add(sig.tech, f.Path)
				}
			}
		}
	}

	// Import/content-based detection
	for _, f := range repo.Files {
		for _, line := range f.Lines {
			for _, sig := range importFrameworkSignals {
				if strings.Contains(line, sig.substr) {
					add(sig.tech, f.Path)
				}
			}
		}
	}

	return techs
}

// DetectInfrastructure returns infrastructure flags from the repo.
func DetectInfrastructure(repo *walker.RepoMap) (hasDocker, hasCompose, hasK8s, hasTF, hasCI bool, ciProviders []string) {
	ciSet := make(map[string]bool)

	for _, f := range repo.Files {
		p := f.Path
		switch {
		case strings.Contains(p, "Dockerfile"):
			hasDocker = true
		case strings.Contains(p, "docker-compose"):
			hasCompose = true
		case strings.Contains(p, "kubernetes") || strings.Contains(p, "/k8s/"):
			hasK8s = true
		case strings.HasSuffix(p, ".tf"):
			hasTF = true
		case strings.Contains(p, ".github/workflows"):
			hasCI = true
			ciSet["github-actions"] = true
		case strings.Contains(p, ".gitlab-ci"):
			hasCI = true
			ciSet["gitlab-ci"] = true
		case strings.Contains(p, ".circleci"):
			hasCI = true
			ciSet["circleci"] = true
		case f.Name == "Jenkinsfile":
			hasCI = true
			ciSet["jenkins"] = true
		case f.Name == ".travis.yml":
			hasCI = true
			ciSet["travis-ci"] = true
		}
	}

	for ci := range ciSet {
		ciProviders = append(ciProviders, ci)
	}
	return
}
