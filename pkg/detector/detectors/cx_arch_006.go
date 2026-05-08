package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type DeadCodeDetector struct{}

func (d *DeadCodeDetector) ID() string       { return "CX-ARCH-006" }
func (d *DeadCodeDetector) Name() string     { return "Dead Code" }
func (d *DeadCodeDetector) Category() string { return detector.CategoryArchitecture }
func (d *DeadCodeDetector) Severity() string { return detector.SeverityLow }

func (d *DeadCodeDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}
		sc := parser.NewScanner(f.Path, f.Lines)

		for i, line := range f.Lines {
			trimmed := strings.TrimSpace(line)

			// Check TODO/FIXME BEFORE skipping — they are comments we want
			if isDeadCodeComment(trimmed) {
				issues = append(issues, detector.Issue{
					Code: d.ID(), Title: d.Name(),
					Message: "TODO/FIXME/HACK comment suggests unfinished or disabled code",
					File:    f.Path, Line: i + 1,
					Severity: detector.SeverityInfo, Confidence: detector.ConfidenceHigh,
					Category: d.Category(), Snippet: trimmed, Context: sc.Context(i+1, 2),
				})
				continue
			}

			// Check for commented-out code
			if parser.IsComment(trimmed, f.Language) {
				if isCommentedOutCode(trimmed, f.Language) {
					issues = append(issues, detector.Issue{
						Code: d.ID(), Title: d.Name(),
						Message: "Commented-out code block — remove and rely on git history",
						File:    f.Path, Line: i + 1,
						Severity: d.Severity(), Confidence: detector.ConfidenceMedium,
						Category: d.Category(), Snippet: trimmed, Context: sc.Context(i+1, 2),
					})
				}
				continue
			}

			// Unreachable code after return/panic/exit
			if isUnreachable(trimmed, f.Language) && i+1 < len(f.Lines) {
				next := strings.TrimSpace(f.Lines[i+1])
				if next != "" && next != "}" && next != ")" &&
					!parser.IsComment(next, f.Language) &&
					!strings.HasPrefix(next, "case ") && !strings.HasPrefix(next, "default:") {
					issues = append(issues, detector.Issue{
						Code: d.ID(), Title: d.Name(),
						Message: "Code after return/panic/exit is unreachable",
						File:    f.Path, Line: i + 2,
						Severity: detector.SeverityMedium, Confidence: detector.ConfidenceHigh,
						Category: d.Category(), Snippet: next, Context: sc.Context(i+1, 2),
					})
				}
			}
		}
	}
	return issues
}

func isUnreachable(line, lang string) bool {
	switch lang {
	case parser.LangGo:
		return line == "return" || strings.HasPrefix(line, "return ") ||
			strings.HasPrefix(line, "panic(") || strings.Contains(line, "os.Exit(")
	case parser.LangJavaScript, parser.LangTypeScript:
		return line == "return" || strings.HasPrefix(line, "return ") || strings.HasPrefix(line, "throw ")
	case parser.LangPython:
		return strings.HasPrefix(line, "return ") || strings.HasPrefix(line, "raise ") || strings.HasPrefix(line, "sys.exit(")
	}
	return false
}

func isDeadCodeComment(line string) bool {
	upper := strings.ToUpper(line)
	for _, marker := range []string{"// TODO", "// FIXME", "// HACK", "// XXX", "# TODO", "# FIXME", "# HACK"} {
		if strings.HasPrefix(upper, strings.ToUpper(marker)) {
			return true
		}
	}
	return false
}

func isCommentedOutCode(line, lang string) bool {
	var prefix string
	switch lang {
	case parser.LangGo, parser.LangJavaScript, parser.LangTypeScript, parser.LangJava, parser.LangRust:
		prefix = "//"
	case parser.LangPython, parser.LangRuby, parser.LangShell:
		prefix = "#"
	default:
		return false
	}
	if !strings.HasPrefix(line, prefix) {
		return false
	}
	content := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	for _, sig := range []string{"func ", "def ", "const ", "var ", "let ", "class ", "import ", "if ", "for ", "return "} {
		if strings.Contains(content, sig) && len(content) > 20 {
			return true
		}
	}
	return false
}
