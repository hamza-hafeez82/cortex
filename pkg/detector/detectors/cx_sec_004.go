package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type PathTraversalDetector struct{}

func (d *PathTraversalDetector) ID() string       { return "CX-SEC-004" }
func (d *PathTraversalDetector) Name() string     { return "Path Traversal" }
func (d *PathTraversalDetector) Category() string { return detector.CategorySecurity }
func (d *PathTraversalDetector) Severity() string { return detector.SeverityHigh }

var fileOps = []string{
	"os.Open(", "os.ReadFile(", "os.WriteFile(", "ioutil.ReadFile(", "ioutil.WriteFile(",
	"open(", "read_file(", "write_file(", "readFileSync(", "writeFileSync(",
	"fs.readFile(", "fs.writeFile(", "fs.readFileSync(", "fs.writeFileSync(",
	"filepath.Join(", "path.join(", "path.resolve(",
	"new File(", "FileInputStream(", "FileOutputStream(",
}

var traversalIndicators = []string{
	"req.", "request.", "params.", "query.", "body.", "args.",
	"input", "user", "+", "format(", "sprintf", "Sprintf",
}

var safePathPatterns = []string{
	"filepath.Clean(", "path.Clean(", "filepath.Abs(",
	"strings.Contains(path, \"..\")", `"../`, `".."`,
	"sanitize", "validate", "allowlist",
}

func (d *PathTraversalDetector) Run(ctx *detector.ScanContext) []detector.Issue {
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

			hasFileOp := false
			for _, op := range fileOps {
				if strings.Contains(trimmed, op) {
					hasFileOp = true
					break
				}
			}
			if !hasFileOp {
				continue
			}

			hasUserInput := false
			for _, pat := range traversalIndicators {
				if strings.Contains(trimmed, pat) {
					hasUserInput = true
					break
				}
			}
			if !hasUserInput {
				continue
			}

			isSafe := false
			for _, safe := range safePathPatterns {
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
				Message:    "File path constructed from user input — validate and sanitize paths to prevent directory traversal",
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
