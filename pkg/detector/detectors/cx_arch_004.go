package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/internal/walker"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// MissingErrorHandlingDetector finds ignored errors and unhandled promises.
type MissingErrorHandlingDetector struct{}

func (d *MissingErrorHandlingDetector) ID() string       { return "CX-ARCH-004" }
func (d *MissingErrorHandlingDetector) Name() string     { return "Missing Error Handling" }
func (d *MissingErrorHandlingDetector) Category() string { return detector.CategoryArchitecture }
func (d *MissingErrorHandlingDetector) Severity() string { return detector.SeverityHigh }

func (d *MissingErrorHandlingDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue
	for _, f := range ctx.Repo.Files {
		if len(f.Lines) == 0 {
			continue
		}
		switch f.Language {
		case parser.LangGo:
			issues = append(issues, checkGoErrors(f)...)
		case parser.LangJavaScript, parser.LangTypeScript:
			issues = append(issues, checkJSErrors(f)...)
		case parser.LangPython:
			issues = append(issues, checkPythonErrors(f)...)
		}
	}
	return issues
}

func checkGoErrors(f *walker.FileNode) []detector.Issue {
	var issues []detector.Issue
	sc := parser.NewScanner(f.Path, f.Lines)
	for i, line := range f.Lines {
		trimmed := strings.TrimSpace(line)
		if parser.IsComment(trimmed, parser.LangGo) {
			continue
		}
		if strings.Contains(trimmed, ", _") && strings.Contains(trimmed, ":=") && endsWithErrCall(trimmed) {
			issues = append(issues, detector.Issue{
				Code: "CX-ARCH-004", Title: "Missing Error Handling",
				Message: "Error return value discarded with _ — handle or log the error",
				File:    f.Path, Line: i + 1,
				Severity: detector.SeverityHigh, Confidence: detector.ConfidenceMedium,
				Category: detector.CategoryArchitecture,
				Snippet:  trimmed, Context: sc.Context(i+1, 3),
			})
		}
	}
	return issues
}

func checkJSErrors(f *walker.FileNode) []detector.Issue {
	var issues []detector.Issue
	sc := parser.NewScanner(f.Path, f.Lines)
	for i, line := range f.Lines {
		trimmed := strings.TrimSpace(line)
		if parser.IsComment(trimmed, f.Language) {
			continue
		}
		if strings.Contains(trimmed, ".then(") && !strings.Contains(trimmed, ".catch(") {
			hasCatch := false
			for j := i + 1; j < len(f.Lines) && j < i+5; j++ {
				if strings.Contains(f.Lines[j], ".catch(") {
					hasCatch = true
					break
				}
			}
			if !hasCatch {
				issues = append(issues, detector.Issue{
					Code: "CX-ARCH-004", Title: "Missing Error Handling",
					Message: "Promise .then() without .catch() — unhandled rejection will silently fail",
					File:    f.Path, Line: i + 1,
					Severity: detector.SeverityHigh, Confidence: detector.ConfidenceMedium,
					Category: detector.CategoryArchitecture,
					Snippet:  trimmed, Context: sc.Context(i+1, 3),
				})
			}
		}
	}
	return issues
}

func checkPythonErrors(f *walker.FileNode) []detector.Issue {
	var issues []detector.Issue
	sc := parser.NewScanner(f.Path, f.Lines)
	for i, line := range f.Lines {
		trimmed := strings.TrimSpace(line)
		if parser.IsComment(trimmed, parser.LangPython) {
			continue
		}
		if trimmed == "except:" || (strings.HasPrefix(trimmed, "except Exception") && strings.HasSuffix(trimmed, ":")) {
			nextLine := ""
			if i+1 < len(f.Lines) {
				nextLine = strings.TrimSpace(f.Lines[i+1])
			}
			if nextLine == "pass" || nextLine == "" {
				issues = append(issues, detector.Issue{
					Code: "CX-ARCH-004", Title: "Missing Error Handling",
					Message: "Bare except clause swallows all exceptions — catch specific exception types",
					File:    f.Path, Line: i + 1,
					Severity: detector.SeverityHigh, Confidence: detector.ConfidenceHigh,
					Category: detector.CategoryArchitecture,
					Snippet:  trimmed, Context: sc.Context(i+1, 3),
				})
			}
		}
	}
	return issues
}

func endsWithErrCall(line string) bool {
	for _, fn := range []string{"Open(", "Read(", "Write(", "Close(", "Exec(", "Query(", "Scan(", "Marshal(", "Unmarshal(", "Parse("} {
		if strings.Contains(line, fn) {
			return true
		}
	}
	return false
}
