package detectors

import (
	"strings"
	"unicode"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/internal/walker"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// NamingConventionsDetector flags variables using the wrong case convention for their language.
type NamingConventionsDetector struct{}

func (d *NamingConventionsDetector) ID() string       { return "CX-ARCH-008" }
func (d *NamingConventionsDetector) Name() string     { return "Inconsistent Naming" }
func (d *NamingConventionsDetector) Category() string { return detector.CategoryArchitecture }
func (d *NamingConventionsDetector) Severity() string { return detector.SeverityLow }

func (d *NamingConventionsDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue
	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}
		issues = append(issues, checkNaming(f)...)
	}
	return issues
}

func checkNaming(f *walker.FileNode) []detector.Issue {
	var issues []detector.Issue
	sc := parser.NewScanner(f.Path, f.Lines)

	for i, line := range f.Lines {
		trimmed := strings.TrimSpace(line)
		if parser.IsComment(trimmed, f.Language) {
			continue
		}

		name := extractVarNameFromLine(trimmed, f.Language)
		if name == "" || len(name) <= 1 {
			continue
		}

		switch f.Language {
		case parser.LangJavaScript, parser.LangTypeScript:
			// JS/TS: should be camelCase, not snake_case
			if isSnakeCaseVar(name) && !isAllCapsConst(name) {
				issues = append(issues, detector.Issue{
					Code: "CX-ARCH-008", Title: "Inconsistent Naming",
					Message: "'" + name + "' uses snake_case — JavaScript/TypeScript convention is camelCase",
					File:    f.Path, Line: i + 1,
					Severity: detector.SeverityLow, Confidence: detector.ConfidenceMedium,
					Category: detector.CategoryArchitecture,
					Snippet:  trimmed, Context: sc.Context(i+1, 2),
				})
			}
		case parser.LangPython:
			// Python: should be snake_case, not camelCase (PEP 8)
			if isCamelCaseVar(name) {
				issues = append(issues, detector.Issue{
					Code: "CX-ARCH-008", Title: "Inconsistent Naming",
					Message: "'" + name + "' uses camelCase — Python convention is snake_case (PEP 8)",
					File:    f.Path, Line: i + 1,
					Severity: detector.SeverityLow, Confidence: detector.ConfidenceMedium,
					Category: detector.CategoryArchitecture,
					Snippet:  trimmed, Context: sc.Context(i+1, 2),
				})
			}
		case parser.LangGo:
			// Go: should be camelCase, not snake_case for locals
			if isSnakeCaseVar(name) && !isAllCapsConst(name) {
				issues = append(issues, detector.Issue{
					Code: "CX-ARCH-008", Title: "Inconsistent Naming",
					Message: "'" + name + "' uses snake_case — Go convention is camelCase for local variables",
					File:    f.Path, Line: i + 1,
					Severity: detector.SeverityLow, Confidence: detector.ConfidenceLow,
					Category: detector.CategoryArchitecture,
					Snippet:  trimmed, Context: sc.Context(i+1, 2),
				})
			}
		}
	}
	return issues
}

func extractVarNameFromLine(line, lang string) string {
	switch lang {
	case parser.LangJavaScript, parser.LangTypeScript:
		for _, prefix := range []string{"const ", "let ", "var "} {
			if strings.HasPrefix(line, prefix) {
				rest := strings.TrimPrefix(line, prefix)
				parts := strings.FieldsFunc(rest, func(r rune) bool {
					return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
				})
				if len(parts) > 0 {
					return parts[0]
				}
			}
		}
	case parser.LangPython:
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "def ") && !strings.HasPrefix(line, "class ") {
			name := strings.TrimSpace(strings.SplitN(line, "=", 2)[0])
			if idx := strings.Index(name, ":"); idx >= 0 {
				name = strings.TrimSpace(name[:idx])
			}
			return name
		}
	case parser.LangGo:
		if strings.Contains(line, ":=") {
			parts := strings.SplitN(line, ":=", 2)
			names := strings.Split(parts[0], ",")
			if len(names) > 0 {
				return strings.TrimSpace(names[0])
			}
		}
	}
	return ""
}

func isSnakeCaseVar(s string) bool {
	return strings.Contains(s, "_") && s == strings.ToLower(s)
}

func isCamelCaseVar(s string) bool {
	if len(s) == 0 || strings.Contains(s, "_") {
		return false
	}
	for _, r := range s[1:] {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func isAllCapsConst(s string) bool {
	return s == strings.ToUpper(s) && strings.Contains(s, "_")
}
