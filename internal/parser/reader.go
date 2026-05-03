package parser

import (
	"strings"
)

// Position identifies an exact location within a file.
type Position struct {
	File   string // relative file path
	Line   int    // 1-indexed line number
	Column int    // 1-indexed column (0 if unknown)
}

// LineMatch represents a line that matched a search criterion.
type LineMatch struct {
	Position Position
	Content  string // raw line content (trimmed)
	Original string // raw line content (untrimmed)
}

// Scanner provides pattern-matching operations over a file's lines.
// It is the primary interface detectors use to inspect source code.
type Scanner struct {
	path  string
	lines []string
}

// NewScanner creates a Scanner for the given file path and line content.
func NewScanner(path string, lines []string) *Scanner {
	return &Scanner{path: path, lines: lines}
}

// Lines returns all lines in the file.
func (s *Scanner) Lines() []string {
	return s.lines
}

// LineCount returns the total number of lines.
func (s *Scanner) LineCount() int {
	return len(s.lines)
}

// Line returns the content of the 1-indexed line number.
// Returns empty string if the line number is out of range.
func (s *Scanner) Line(n int) string {
	if n < 1 || n > len(s.lines) {
		return ""
	}
	return s.lines[n-1]
}

// ContainsAny returns all lines that contain any of the given substrings.
// The search is case-sensitive. Line numbers are 1-indexed.
func (s *Scanner) ContainsAny(substrings ...string) []LineMatch {
	var matches []LineMatch
	for i, line := range s.lines {
		for _, sub := range substrings {
			if strings.Contains(line, sub) {
				matches = append(matches, LineMatch{
					Position: Position{File: s.path, Line: i + 1},
					Content:  strings.TrimSpace(line),
					Original: line,
				})
				break // one match per line is enough
			}
		}
	}
	return matches
}

// ContainsAnyCI is like ContainsAny but case-insensitive.
func (s *Scanner) ContainsAnyCI(substrings ...string) []LineMatch {
	lower := make([]string, len(substrings))
	for i, s := range substrings {
		lower[i] = strings.ToLower(s)
	}

	var matches []LineMatch
	for i, line := range s.lines {
		lineLower := strings.ToLower(line)
		for _, sub := range lower {
			if strings.Contains(lineLower, sub) {
				matches = append(matches, LineMatch{
					Position: Position{File: s.path, Line: i + 1},
					Content:  strings.TrimSpace(line),
					Original: line,
				})
				break
			}
		}
	}
	return matches
}

// ContainsAllOnLine returns lines that contain ALL of the given substrings.
// Useful for narrowing matches, e.g. a line with both "password" and "=".
func (s *Scanner) ContainsAllOnLine(substrings ...string) []LineMatch {
	var matches []LineMatch
	for i, line := range s.lines {
		allFound := true
		for _, sub := range substrings {
			if !strings.Contains(line, sub) {
				allFound = false
				break
			}
		}
		if allFound {
			matches = append(matches, LineMatch{
				Position: Position{File: s.path, Line: i + 1},
				Content:  strings.TrimSpace(line),
				Original: line,
			})
		}
	}
	return matches
}

// Context returns up to n lines before and after the given 1-indexed line number.
// Useful for building AI explanation context.
func (s *Scanner) Context(lineNum, n int) []string {
	start := lineNum - 1 - n
	end := lineNum - 1 + n

	if start < 0 {
		start = 0
	}
	if end >= len(s.lines) {
		end = len(s.lines) - 1
	}

	result := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		result = append(result, s.lines[i])
	}
	return result
}

// IsComment returns true if the given line is a comment in the most common
// languages. Used to reduce false positives in detectors.
func IsComment(line, lang string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Single-line comment prefixes common to most languages
	if strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "--") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") {
		return true
	}

	// Python/Ruby docstrings
	if lang == LangPython || lang == LangRuby {
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			return true
		}
	}

	return false
}

// StripInlineComment removes the comment portion from the end of a line.
// For example: `password = "secret" // TODO` → `password = "secret" `
func StripInlineComment(line string) string {
	// Handle // style
	if idx := strings.Index(line, "//"); idx >= 0 {
		// Make sure it's not inside a string (simple heuristic: count quotes)
		before := line[:idx]
		if strings.Count(before, `"`)%2 == 0 {
			return before
		}
	}
	return line
}
