package detectors_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hamza-hafeez82/cortex/internal/walker"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
	"github.com/hamza-hafeez82/cortex/pkg/detector/detectors"
)

func makeArchCtx(t *testing.T, files map[string]string) *detector.ScanContext {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w := walker.New(walker.DefaultOptions())
	repo, err := w.Walk(root)
	if err != nil {
		t.Fatal(err)
	}
	return &detector.ScanContext{Repo: repo}
}

// ── CX-ARCH-001: God File ─────────────────────────────────────────────────────

func TestGodFile_Detects(t *testing.T) {
	// Build a file with 500+ lines and 10+ functions
	var sb strings.Builder
	for i := 0; i < 15; i++ {
		sb.WriteString("func doSomething() {\n")
		for j := 0; j < 35; j++ {
			sb.WriteString("  x := 1\n")
		}
		sb.WriteString("}\n")
	}

	ctx := makeArchCtx(t, map[string]string{"app.go": sb.String()})
	d := &detectors.GodFileDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected god file to be detected")
	}
}

func TestGodFile_SmallFileIgnored(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"small.go": "package main\n\nfunc main() {}\n",
	})
	d := &detectors.GodFileDetector{}
	issues := d.Run(ctx)
	if len(issues) != 0 {
		t.Errorf("small file should not trigger, got %d issues", len(issues))
	}
}

// ── CX-ARCH-002: Circular Dependencies ───────────────────────────────────────

func TestCircularDeps_DetectsRelativeImport(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"app/models.py": `from app import services
class User:
    pass
`,
		"app/services.py": `from app import models
def get_user():
    pass
`,
	})
	d := &detectors.CircularDepsDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected circular dependency to be detected")
	}
}

func TestCircularDeps_NoCycleClean(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"pkg/a/a.go": `package a
import "fmt"
`,
		"pkg/b/b.go": `package b
import "strings"
`,
	})
	d := &detectors.CircularDepsDetector{}
	issues := d.Run(ctx)
	if len(issues) != 0 {
		t.Errorf("no cycle should produce no issues, got %d", len(issues))
	}
}

// ── CX-ARCH-003: Deep Nesting ─────────────────────────────────────────────────

func TestDeepNesting_Detects(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"handler.go": `package main
func process() {
  if a {
    for i := 0; i < 10; i++ {
      if b {
        switch c {
          case 1: {
            if d {
              doSomething()
            }
          }
        }
      }
    }
  }
}
`,
	})
	d := &detectors.DeepNestingDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected deep nesting to be detected")
	}
}

func TestDeepNesting_CleanCode(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"clean.go": `package main
func process() {
  if err != nil {
    return err
  }
  doSomething()
}
`,
	})
	d := &detectors.DeepNestingDetector{}
	issues := d.Run(ctx)
	if len(issues) != 0 {
		t.Errorf("flat code should not trigger, got %d issues", len(issues))
	}
}

// ── CX-ARCH-004: Missing Error Handling ──────────────────────────────────────

func TestMissingErrorHandling_GoBlankErr(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"db.go": `package main
func query() {
  rows, _ := db.Query("SELECT * FROM users")
  defer rows.Close()
}
`,
	})
	d := &detectors.MissingErrorHandlingDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected discarded error to be detected")
	}
}

func TestMissingErrorHandling_JSPromiseNoCatch(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"api.js": `async function fetchUser(id) {
  fetch('/api/users/' + id)
    .then(res => res.json())
    .then(data => console.log(data))
}
`,
	})
	d := &detectors.MissingErrorHandlingDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected missing .catch() to be detected")
	}
}

func TestMissingErrorHandling_PythonBareExcept(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"handler.py": `def process():
    try:
        risky_operation()
    except:
        pass
`,
	})
	d := &detectors.MissingErrorHandlingDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected bare except with pass to be detected")
	}
}

// ── CX-ARCH-005: Magic Numbers ────────────────────────────────────────────────

func TestMagicNumbers_Detects(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"auth.go": `package main
func isExpired(age int) bool {
  return age > 86400
}
`,
	})
	d := &detectors.MagicNumbersDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected magic number 86400 to be detected")
	}
}

func TestMagicNumbers_ConstIgnored(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"auth.go": `package main
const MaxSessionAge = 86400
func isExpired(age int) bool {
  return age > MaxSessionAge
}
`,
	})
	d := &detectors.MagicNumbersDetector{}
	issues := d.Run(ctx)
	// const declaration line should not be flagged
	for _, issue := range issues {
		if strings.Contains(issue.Snippet, "const ") {
			t.Error("const declaration should not trigger magic number detector")
		}
	}
}

// ── CX-ARCH-006: Dead Code ────────────────────────────────────────────────────

func TestDeadCode_DetectsUnreachable(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"handler.go": `package main
func getUser() *User {
  return nil
  log.Println("this never runs")
}
`,
	})
	d := &detectors.DeadCodeDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected unreachable code after return to be detected")
	}
}

func TestDeadCode_DetectsTODO(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"service.go": `package main
func process() {
  // TODO: implement retry logic
  doSomething()
}
`,
	})
	d := &detectors.DeadCodeDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected TODO comment to be detected")
	}
}

// ── CX-ARCH-007: Missing Tests ────────────────────────────────────────────────

func TestMissingTests_Detects(t *testing.T) {
	// Large file with no test counterpart
	var sb strings.Builder
	for i := 0; i < 35; i++ {
		sb.WriteString("func doWork() { x := 1 }\n")
	}
	ctx := makeArchCtx(t, map[string]string{
		"service.go": sb.String(),
	})
	d := &detectors.MissingTestsDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected missing test file to be detected")
	}
}

func TestMissingTests_HasTestFile(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 35; i++ {
		sb.WriteString("func doWork() { x := 1 }\n")
	}
	ctx := makeArchCtx(t, map[string]string{
		"service.go":      sb.String(),
		"service_test.go": "package main\nfunc TestDoWork(t *testing.T) {}\n",
	})
	d := &detectors.MissingTestsDetector{}
	issues := d.Run(ctx)
	// service.go should not trigger since service_test.go exists
	for _, issue := range issues {
		if strings.Contains(issue.File, "service.go") {
			t.Error("service.go should not trigger when service_test.go exists")
		}
	}
}

// ── CX-ARCH-008: Naming Conventions ──────────────────────────────────────────

func TestNamingConventions_JSSnakeCase(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"app.js": `const user_name = "john"
const first_name = "doe"
`,
	})
	d := &detectors.NamingConventionsDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected snake_case in JS to be detected")
	}
}

func TestNamingConventions_PythonCamelCase(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"models.py": `userName = "john"
firstName = "doe"
`,
	})
	d := &detectors.NamingConventionsDetector{}
	issues := d.Run(ctx)
	if len(issues) == 0 {
		t.Error("expected camelCase in Python to be detected")
	}
}

func TestNamingConventions_CorrectJS(t *testing.T) {
	ctx := makeArchCtx(t, map[string]string{
		"app.js": `const userName = "john"
const firstName = "doe"
const MAX_RETRIES = 3
`,
	})
	d := &detectors.NamingConventionsDetector{}
	issues := d.Run(ctx)
	if len(issues) != 0 {
		t.Errorf("correct camelCase JS should not trigger, got %d issues", len(issues))
	}
}
