package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type SQLInjectionDetector struct{}

func (d *SQLInjectionDetector) ID() string       { return "CX-SEC-002" }
func (d *SQLInjectionDetector) Name() string     { return "SQL Injection" }
func (d *SQLInjectionDetector) Category() string { return detector.CategorySecurity }
func (d *SQLInjectionDetector) Severity() string { return detector.SeverityHigh }

var sqlKeywords = []string{"SELECT ", "INSERT INTO", "UPDATE ", "DELETE FROM", "DROP TABLE", "CREATE TABLE"}
var concatOperators = []string{`" +`, `' +`, `"+`, `'+`, `" .`, `' .`, `%.`, `%s`, `%v`, `format(`, `f"`, `f'`}
var safeQueryPatterns = []string{"$1", "$2", "?", ":name", "@param", "Prepare(", "prepare(", "parameterized"}

func (d *SQLInjectionDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}
		sc := parser.NewScanner(f.Path, f.Lines)

		for i, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			if parser.IsComment(trimmed, f.Language) {
				continue
			}
			lineUpper := strings.ToUpper(trimmed)

			hasSQLKeyword := false
			for _, kw := range sqlKeywords {
				if strings.Contains(lineUpper, kw) {
					hasSQLKeyword = true
					break
				}
			}
			if !hasSQLKeyword {
				continue
			}

			hasConcatenation := false
			for _, op := range concatOperators {
				if strings.Contains(trimmed, op) {
					hasConcatenation = true
					break
				}
			}
			if !hasConcatenation {
				continue
			}

			isSafe := false
			for _, safe := range safeQueryPatterns {
				if strings.Contains(trimmed, safe) {
					isSafe = true
					break
				}
			}
			if isSafe {
				continue
			}

			issues = append(issues, detector.Issue{
				Code:       d.ID(),
				Title:      d.Name(),
				Message:    "SQL query built with string concatenation — use parameterized queries to prevent SQL injection",
				File:       f.Path,
				Line:       i + 1,
				Severity:   d.Severity(),
				Confidence: detector.ConfidenceMedium,
				Category:   d.Category(),
				Snippet:    trimmed,
				Context:    sc.Context(i+1, 3),
			})
		}
	}
	return issues
}
