package parser_test

import (
	"testing"

	"github.com/hamza-hafeez82/cortex/internal/parser"
)

func newScanner(lines ...string) *parser.Scanner {
	return parser.NewScanner("test/file.go", lines)
}

func TestScannerLineCount(t *testing.T) {
	s := newScanner("a", "b", "c")
	if s.LineCount() != 3 {
		t.Errorf("expected 3, got %d", s.LineCount())
	}
}

func TestScannerLine(t *testing.T) {
	s := newScanner("first", "second", "third")

	if s.Line(1) != "first" {
		t.Errorf("Line(1) = %q, want %q", s.Line(1), "first")
	}
	if s.Line(3) != "third" {
		t.Errorf("Line(3) = %q, want %q", s.Line(3), "third")
	}
	if s.Line(0) != "" {
		t.Error("Line(0) should return empty string")
	}
	if s.Line(99) != "" {
		t.Error("Line(99) should return empty string for out-of-range")
	}
}

func TestScannerContainsAny(t *testing.T) {
	s := newScanner(
		`password := "secret123"`,
		`username := "admin"`,
		`apiKey = "sk-1234567890"`,
	)

	matches := s.ContainsAny("password", "apiKey")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Position.Line != 1 {
		t.Errorf("first match should be line 1, got %d", matches[0].Position.Line)
	}
	if matches[1].Position.Line != 3 {
		t.Errorf("second match should be line 3, got %d", matches[1].Position.Line)
	}
}

func TestScannerContainsAnyCI(t *testing.T) {
	s := newScanner(
		`PASSWORD := "secret"`,
		`username := "admin"`,
		`Api_Key = "value"`,
	)

	matches := s.ContainsAnyCI("password", "api_key")
	if len(matches) != 2 {
		t.Errorf("expected 2 case-insensitive matches, got %d", len(matches))
	}
}

func TestScannerContainsAllOnLine(t *testing.T) {
	s := newScanner(
		`password = "secret"`,
		`// password is set elsewhere`,
		`db_pass = getenv("DB_PASS")`,
	)

	// Only line 1 has both "password" and "="  and a literal value
	matches := s.ContainsAllOnLine("password", `"`)
	if len(matches) != 1 {
		t.Errorf("expected 1 match (both substrings on same line), got %d", len(matches))
	}
}

func TestScannerContext(t *testing.T) {
	s := newScanner("L1", "L2", "L3", "L4", "L5", "L6", "L7")

	ctx := s.Context(4, 2) // line 4, 2 lines context each side
	// Should return lines 2-6
	if len(ctx) != 5 {
		t.Errorf("expected 5 context lines, got %d", len(ctx))
	}
	if ctx[0] != "L2" {
		t.Errorf("context[0] = %q, want L2", ctx[0])
	}
	if ctx[4] != "L6" {
		t.Errorf("context[4] = %q, want L6", ctx[4])
	}
}

func TestScannerContextAtBoundaries(t *testing.T) {
	s := newScanner("L1", "L2", "L3")

	// Context at start — should not underflow
	ctx := s.Context(1, 3)
	if len(ctx) != 3 {
		t.Errorf("expected 3 lines at start boundary, got %d", len(ctx))
	}

	// Context at end — should not overflow
	ctx = s.Context(3, 3)
	if len(ctx) != 3 {
		t.Errorf("expected 3 lines at end boundary, got %d", len(ctx))
	}
}

func TestIsComment(t *testing.T) {
	tests := []struct {
		line     string
		lang     string
		expected bool
	}{
		{"// this is a comment", parser.LangGo, true},
		{"# python comment", parser.LangPython, true},
		{"/* block comment */", parser.LangJavaScript, true},
		{" * continuation", parser.LangJava, true},
		{"-- SQL comment", parser.LangSQL, true},
		{`password = "secret"`, parser.LangGo, false},
		{"", parser.LangGo, false},
	}

	for _, tt := range tests {
		got := parser.IsComment(tt.line, tt.lang)
		if got != tt.expected {
			t.Errorf("IsComment(%q, %q) = %v, want %v", tt.line, tt.lang, got, tt.expected)
		}
	}
}
