package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// MissingTestsDetector flags source files that have no corresponding test file.
type MissingTestsDetector struct{}

func (d *MissingTestsDetector) ID() string       { return "CX-ARCH-007" }
func (d *MissingTestsDetector) Name() string     { return "Missing Tests" }
func (d *MissingTestsDetector) Category() string { return detector.CategoryArchitecture }
func (d *MissingTestsDetector) Severity() string { return detector.SeverityMedium }

func (d *MissingTestsDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	// Build a set of all test file paths for fast lookup
	testFiles := make(map[string]bool)
	for _, f := range ctx.Repo.Files {
		if isTestFile(f.Path, f.Name, f.Language) {
			testFiles[f.Path] = true
		}
	}

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}
		if isTestFile(f.Path, f.Name, f.Language) {
			continue
		}
		// Skip very small files (< 30 lines) — not worth testing individually
		if f.LineCount < 30 {
			continue
		}
		// Skip entry points and config files
		if isEntryPoint(f.Name) {
			continue
		}

		if !hasTestCounterpart(f.Path, f.Name, f.Language, testFiles) {
			issues = append(issues, detector.Issue{
				Code:       d.ID(),
				Title:      d.Name(),
				Message:    "No test file found for " + f.Path + " — add unit tests to ensure correctness",
				File:       f.Path,
				Line:       0,
				Severity:   d.Severity(),
				Confidence: detector.ConfidenceMedium,
				Category:   d.Category(),
				Snippet:    f.Name,
			})
		}
	}

	return issues
}

// isTestFile returns true if the file is itself a test file.
func isTestFile(path, name, lang string) bool {
	switch lang {
	case parser.LangGo:
		return strings.HasSuffix(name, "_test.go")
	case parser.LangJavaScript, parser.LangTypeScript:
		return strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") ||
			strings.Contains(path, "/__tests__/") || strings.Contains(path, "/test/") ||
			strings.Contains(path, "/tests/")
	case parser.LangPython:
		return strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test.py") ||
			strings.Contains(path, "/tests/") || strings.Contains(path, "/test/")
	case parser.LangRust:
		// Rust tests are inline, but test modules in separate files
		return strings.Contains(path, "/tests/")
	case parser.LangJava:
		return strings.HasSuffix(name, "Test.java") || strings.Contains(path, "/test/")
	}
	return false
}

// hasTestCounterpart checks if a corresponding test file exists.
func hasTestCounterpart(path, name, lang string, testFiles map[string]bool) bool {
	dir := ""
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		dir = path[:idx+1]
	}

	var candidates []string

	switch lang {
	case parser.LangGo:
		base := strings.TrimSuffix(name, ".go")
		candidates = []string{dir + base + "_test.go"}

	case parser.LangJavaScript, parser.LangTypeScript:
		for _, ext := range []string{".js", ".ts", ".jsx", ".tsx"} {
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(name, ".js"), ".ts"), ".jsx"), ".tsx")
			candidates = append(candidates,
				dir+base+".test"+ext,
				dir+base+".spec"+ext,
				dir+"__tests__/"+base+".test"+ext,
			)
		}

	case parser.LangPython:
		base := strings.TrimSuffix(name, ".py")
		candidates = []string{
			dir + "test_" + base + ".py",
			dir + base + "_test.py",
			"tests/test_" + base + ".py",
			"test/test_" + base + ".py",
		}

	case parser.LangJava:
		base := strings.TrimSuffix(name, ".java")
		candidates = []string{dir + base + "Test.java"}
	}

	for _, c := range candidates {
		if testFiles[c] {
			return true
		}
	}

	// Also check if any test file in the same directory references this file's name
	base := strings.TrimSuffix(name, ".go")
	base = strings.TrimSuffix(base, ".py")
	base = strings.TrimSuffix(base, ".js")
	base = strings.TrimSuffix(base, ".ts")

	for testPath := range testFiles {
		if strings.HasPrefix(testPath, dir) && strings.Contains(testPath, base) {
			return true
		}
	}

	return false
}

func isEntryPoint(name string) bool {
	entryPoints := []string{
		"main.go", "main.py", "index.js", "index.ts",
		"app.js", "app.ts", "server.js", "server.ts",
		"manage.py", "wsgi.py", "asgi.py",
	}
	for _, ep := range entryPoints {
		if name == ep {
			return true
		}
	}
	return false
}
