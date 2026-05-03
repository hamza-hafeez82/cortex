package walker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hamza-hafeez82/cortex/internal/walker"
)

// makeDir creates a temp directory structure from a map of
// relative path → file content. Returns the root path and a cleanup func.
func makeDir(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("creating dirs for %s: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", rel, err)
		}
	}
	return root
}

func newWalker() *walker.Walker {
	opts := walker.DefaultOptions()
	opts.LoadLines = true
	return walker.New(opts)
}

// TestWalkBasic verifies that a simple repo is scanned correctly.
func TestWalkBasic(t *testing.T) {
	root := makeDir(t, map[string]string{
		"main.go":        "package main\n\nfunc main() {}\n",
		"internal/db.go": "package internal\n",
		"README.md":      "# Hello\n",
	})

	w := newWalker()
	repo, err := w.Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	if repo.TotalFiles != 3 {
		t.Errorf("expected 3 files, got %d", repo.TotalFiles)
	}

	mainNode := repo.ByPath["main.go"]
	if mainNode == nil {
		t.Fatal("main.go not found in repo map")
	}
	if mainNode.Language != "Go" {
		t.Errorf("expected language Go, got %q", mainNode.Language)
	}
	if mainNode.LineCount != 3 {
		t.Errorf("expected 3 lines, got %d", mainNode.LineCount)
	}
}

// TestWalkIgnoresNodeModules verifies that ignored directories are skipped.
func TestWalkIgnoresNodeModules(t *testing.T) {
	root := makeDir(t, map[string]string{
		"index.js":                    "console.log('hello')\n",
		"node_modules/lodash/main.js": "// lodash\n",
		"src/app.js":                  "// app\n",
	})

	repo, err := newWalker().Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	for _, f := range repo.Files {
		if filepath.ToSlash(f.Path) == "node_modules/lodash/main.js" {
			t.Error("node_modules file should have been ignored")
		}
	}

	if repo.TotalFiles != 2 {
		t.Errorf("expected 2 files (node_modules excluded), got %d", repo.TotalFiles)
	}
}

// TestWalkIgnoresGitDir verifies that .git is skipped.
func TestWalkIgnoresGitDir(t *testing.T) {
	root := makeDir(t, map[string]string{
		"main.go":     "package main\n",
		".git/HEAD":   "ref: refs/heads/main\n",
		".git/config": "[core]\n",
	})

	repo, err := newWalker().Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	if repo.TotalFiles != 1 {
		t.Errorf("expected 1 file (.git excluded), got %d", repo.TotalFiles)
	}
}

// TestWalkIgnoresBinaryExtensions verifies binary files are excluded.
func TestWalkIgnoresBinaryExtensions(t *testing.T) {
	root := makeDir(t, map[string]string{
		"main.go":  "package main\n",
		"icon.png": "\x89PNG\r\n",
		"app.exe":  "\x4D\x5A",
		"data.db":  "SQLite",
	})

	repo, err := newWalker().Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	if repo.TotalFiles != 1 {
		t.Errorf("expected only main.go (binaries excluded), got %d files", repo.TotalFiles)
	}
}

// TestWalkEmptyRepo verifies that an empty directory produces an empty RepoMap.
func TestWalkEmptyRepo(t *testing.T) {
	root := t.TempDir()

	repo, err := newWalker().Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	if repo.TotalFiles != 0 {
		t.Errorf("expected 0 files, got %d", repo.TotalFiles)
	}
}

// TestWalkByLanguageIndex verifies the ByLanguage index is populated correctly.
func TestWalkByLanguageIndex(t *testing.T) {
	root := makeDir(t, map[string]string{
		"main.go":    "package main\n",
		"server.go":  "package main\n",
		"index.js":   "const x = 1\n",
		"styles.css": "body { margin: 0 }\n",
	})

	repo, err := newWalker().Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	goFiles := repo.ByLanguage["Go"]
	if len(goFiles) != 2 {
		t.Errorf("expected 2 Go files, got %d", len(goFiles))
	}

	jsFiles := repo.ByLanguage["JavaScript"]
	if len(jsFiles) != 1 {
		t.Errorf("expected 1 JavaScript file, got %d", len(jsFiles))
	}
}

// TestWalkNestedDirs verifies deep directory trees are traversed correctly.
func TestWalkNestedDirs(t *testing.T) {
	root := makeDir(t, map[string]string{
		"a/b/c/d/deep.go": "package d\n",
		"a/b/mid.go":      "package b\n",
		"top.go":          "package main\n",
	})

	repo, err := newWalker().Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	if repo.TotalFiles != 3 {
		t.Errorf("expected 3 files in nested dirs, got %d", repo.TotalFiles)
	}
}

// TestWalkLineContent verifies that line content is loaded correctly.
func TestWalkLineContent(t *testing.T) {
	content := "line one\nline two\nline three\n"
	root := makeDir(t, map[string]string{
		"test.go": content,
	})

	repo, err := newWalker().Walk(root)
	if err != nil {
		t.Fatalf("Walk() error: %v", err)
	}

	node := repo.ByPath["test.go"]
	if node == nil {
		t.Fatal("test.go not found")
	}

	if node.LineCount != 3 {
		t.Errorf("expected 3 lines, got %d", node.LineCount)
	}
	if node.Lines[0] != "line one" {
		t.Errorf("expected 'line one', got %q", node.Lines[0])
	}
	if node.Lines[2] != "line three" {
		t.Errorf("expected 'line three', got %q", node.Lines[2])
	}
}

// TestWalkInvalidRoot verifies that a nonexistent root returns an error.
func TestWalkInvalidRoot(t *testing.T) {
	_, err := newWalker().Walk("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent root, got nil")
	}
}

// TestWalkRootIsFile verifies that passing a file path returns an error.
func TestWalkRootIsFile(t *testing.T) {
	root := makeDir(t, map[string]string{"main.go": "package main\n"})
	filePath := filepath.Join(root, "main.go")

	_, err := newWalker().Walk(filePath)
	if err == nil {
		t.Error("expected error when root is a file, got nil")
	}
}

// BenchmarkWalk measures walker performance on a generated file tree.
func BenchmarkWalk(b *testing.B) {
	// Build a temp repo with ~200 Go files across nested directories
	files := make(map[string]string)
	for i := 0; i < 200; i++ {
		path := filepath.Join("pkg", "module", filepath.FromSlash("sub/file_"+itoa(i)+".go"))
		files[path] = "package sub\n\nfunc Fn() {}\n"
	}
	root := makeDir(&testing.T{}, files)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := newWalker()
		if _, err := w.Walk(root); err != nil {
			b.Fatal(err)
		}
	}
}

func itoa(n int) string {
	return string(rune('0' + n%10))
}
