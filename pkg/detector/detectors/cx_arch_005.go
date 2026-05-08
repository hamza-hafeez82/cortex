package detectors

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// MagicNumbersDetector flags unexplained numeric literals in logic code.
type MagicNumbersDetector struct{}

func (d *MagicNumbersDetector) ID() string       { return "CX-ARCH-005" }
func (d *MagicNumbersDetector) Name() string     { return "Magic Numbers" }
func (d *MagicNumbersDetector) Category() string { return detector.CategoryArchitecture }
func (d *MagicNumbersDetector) Severity() string { return detector.SeverityLow }

// allowedNumbers are values that are always acceptable without a named constant.
var allowedNumbers = map[string]bool{
	"0": true, "1": true, "2": true, "-1": true,
	"0.0": true, "1.0": true, "100": true,
	"true": true, "false": true,
}

// allowedContexts are keywords that make a numeric literal acceptable inline.
var allowedContexts = []string{
	"version", "port", "timeout", "size", "length", "count",
	"index", "http.", "status", "statuscode", "retry",
}

func (d *MagicNumbersDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}

		sc := parser.NewScanner(f.Path, f.Lines)
		fileIssueCount := 0

		for i, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			if parser.IsComment(trimmed, f.Language) {
				continue
			}
			// Skip const/var declarations — these are exactly where numbers belong
			if isConstDeclaration(trimmed, f.Language) {
				continue
			}

			numbers := extractMagicNumbers(trimmed)
			for _, num := range numbers {
				if allowedNumbers[num] {
					continue
				}
				if hasAllowedContext(trimmed) {
					continue
				}

				issues = append(issues, detector.Issue{
					Code:       d.ID(),
					Title:      d.Name(),
					Message:    fmt.Sprintf("Magic number %s — extract into a named constant to clarify intent", num),
					File:       f.Path,
					Line:       i + 1,
					Severity:   d.Severity(),
					Confidence: detector.ConfidenceLow,
					Category:   d.Category(),
					Snippet:    trimmed,
					Context:    sc.Context(i+1, 2),
				})
				fileIssueCount++
				break // one per line is enough
			}

			// Cap issues per file to avoid noise in generated/data files
			if fileIssueCount >= 5 {
				break
			}
		}
	}

	return issues
}

// extractMagicNumbers pulls numeric literals from a line of code.
func extractMagicNumbers(line string) []string {
	var numbers []string
	words := strings.FieldsFunc(line, func(r rune) bool {
		return !unicode.IsDigit(r) && r != '.' && r != '-'
	})

	for _, w := range words {
		w = strings.Trim(w, ".")
		if w == "" || w == "-" {
			continue
		}
		// Must parse as a number
		if _, err := strconv.ParseFloat(w, 64); err != nil {
			continue
		}
		// Skip small numbers
		if f, _ := strconv.ParseFloat(w, 64); f >= -1 && f <= 2 {
			continue
		}
		numbers = append(numbers, w)
	}
	return numbers
}

func isConstDeclaration(line, lang string) bool {
	switch lang {
	case parser.LangGo:
		return strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "var ")
	case parser.LangJavaScript, parser.LangTypeScript:
		return strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") ||
			strings.HasPrefix(line, "var ") || strings.Contains(line, "readonly ")
	case parser.LangPython:
		// Python convention: ALL_CAPS = value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			return name == strings.ToUpper(name) && name != ""
		}
	}
	return false
}

func hasAllowedContext(line string) bool {
	lineLower := strings.ToLower(line)
	for _, ctx := range allowedContexts {
		if strings.Contains(lineLower, ctx) {
			return true
		}
	}
	return false
}
