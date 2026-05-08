package detectors

import (
	"fmt"
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// DeepNestingDetector flags code blocks nested beyond a healthy depth.
type DeepNestingDetector struct{}

func (d *DeepNestingDetector) ID() string       { return "CX-ARCH-003" }
func (d *DeepNestingDetector) Name() string     { return "Deep Nesting" }
func (d *DeepNestingDetector) Category() string { return detector.CategoryArchitecture }
func (d *DeepNestingDetector) Severity() string { return detector.SeverityMedium }

const maxNestingDepth = 4

func (d *DeepNestingDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}

		// Only check brace-based languages
		if !usesBraces(f.Language) {
			continue
		}

		depth := 0
		reported := make(map[int]bool) // avoid multiple reports per deep block

		sc := parser.NewScanner(f.Path, f.Lines)

		for i, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			if parser.IsComment(trimmed, f.Language) {
				continue
			}

			// Count braces on this line
			for _, ch := range line {
				switch ch {
				case '{':
					depth++
				case '}':
					if depth > 0 {
						depth--
					}
				}
			}

			if depth > maxNestingDepth && !reported[depth] {
				reported[depth] = true
				issues = append(issues, detector.Issue{
					Code:       d.ID(),
					Title:      d.Name(),
					Message:    fmt.Sprintf("Nesting depth of %d exceeds recommended maximum of %d — consider extracting logic into named functions", depth, maxNestingDepth),
					File:       f.Path,
					Line:       i + 1,
					Severity:   d.Severity(),
					Confidence: detector.ConfidenceMedium,
					Category:   d.Category(),
					Snippet:    trimmed,
					Context:    sc.Context(i+1, 2),
				})
			}

			// Reset per-depth reporting when we come back up
			if depth <= maxNestingDepth {
				for k := range reported {
					if k > maxNestingDepth {
						delete(reported, k)
					}
				}
			}
		}
	}

	return issues
}

func usesBraces(lang string) bool {
	switch lang {
	case parser.LangGo, parser.LangJavaScript, parser.LangTypeScript,
		parser.LangJava, parser.LangCSharp, parser.LangC, parser.LangCPP,
		parser.LangRust, parser.LangKotlin, parser.LangSwift, parser.LangScala,
		parser.LangPHP:
		return true
	}
	return false
}
