package detectors

import (
	"fmt"
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// GodFileDetector flags files that are too large and carry too many responsibilities.
type GodFileDetector struct{}

func (d *GodFileDetector) ID() string       { return "CX-ARCH-001" }
func (d *GodFileDetector) Name() string     { return "God File" }
func (d *GodFileDetector) Category() string { return detector.CategoryArchitecture }
func (d *GodFileDetector) Severity() string { return detector.SeverityMedium }

const (
	godFileLineThreshold     = 500 // lines
	godFileFunctionThreshold = 10  // distinct function/method definitions
)

// functionKeywords are language-specific function declaration keywords.
var functionKeywords = []string{
	"func ",      // Go
	"function ",  // JS/TS
	"def ",       // Python
	"fn ",        // Rust
	"public ",    // Java/C#
	"private ",   // Java/C#
	"protected ", // Java/C#
	"async ",     // JS/TS
}

func (d *GodFileDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}

		if f.LineCount < godFileLineThreshold {
			continue
		}

		// Count function/method definitions as a proxy for responsibilities
		fnCount := 0
		for _, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			if parser.IsComment(trimmed, f.Language) {
				continue
			}
			for _, kw := range functionKeywords {
				if strings.Contains(trimmed, kw) && strings.Contains(trimmed, "(") {
					fnCount++
					break
				}
			}
		}

		if fnCount < godFileFunctionThreshold {
			continue
		}

		severity := detector.SeverityMedium
		if f.LineCount > 1000 {
			severity = detector.SeverityHigh
		}

		issues = append(issues, detector.Issue{
			Code:       d.ID(),
			Title:      d.Name(),
			Message:    fmt.Sprintf("File has %d lines and ~%d functions — split into smaller, focused modules", f.LineCount, fnCount),
			File:       f.Path,
			Line:       0,
			Severity:   severity,
			Confidence: detector.ConfidenceHigh,
			Category:   d.Category(),
			Snippet:    fmt.Sprintf("%d lines, ~%d functions", f.LineCount, fnCount),
		})
	}

	return issues
}
