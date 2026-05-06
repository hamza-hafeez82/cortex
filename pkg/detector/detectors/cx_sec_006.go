package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type JWTMisconfigDetector struct{}

func (d *JWTMisconfigDetector) ID() string       { return "CX-SEC-006" }
func (d *JWTMisconfigDetector) Name() string     { return "JWT Misconfiguration" }
func (d *JWTMisconfigDetector) Category() string { return detector.CategorySecurity }
func (d *JWTMisconfigDetector) Severity() string { return detector.SeverityHigh }

// Dangerous patterns that indicate JWT misconfigurations.
var jwtDangerousPatterns = []struct {
	pattern string
	message string
}{
	{`"none"`, "JWT algorithm set to 'none' — tokens will not be verified"},
	{`'none'`, "JWT algorithm set to 'none' — tokens will not be verified"},
	{"alg: none", "JWT algorithm set to 'none' — tokens will not be verified"},
	{"algorithm: none", "JWT algorithm set to 'none' — tokens will not be verified"},
	{"algorithms=['none']", "JWT algorithm set to 'none' — tokens will not be verified"},
	{`algorithms=["none"]`, "JWT algorithm set to 'none' — tokens will not be verified"},
	{"verify: false", "JWT verification disabled — tokens are accepted without validation"},
	{"verify_signature: false", "JWT signature verification disabled"},
	{"ignoreExpiration: true", "JWT expiration check disabled — expired tokens will be accepted"},
	{"options={\"verify_signature\": False}", "JWT signature verification disabled"},
	{"jwt.decode(token,", "JWT decoded without algorithm specification — vulnerable to algorithm confusion"},
}

func (d *JWTMisconfigDetector) Run(ctx *detector.ScanContext) []detector.Issue {
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

			for _, pat := range jwtDangerousPatterns {
				if strings.Contains(trimmed, pat.pattern) {
					issues = append(issues, detector.Issue{
						Code:       d.ID(),
						Title:      d.Name(),
						Message:    pat.message,
						File:       f.Path,
						Line:       i + 1,
						Severity:   d.Severity(),
						Confidence: detector.ConfidenceHigh,
						Category:   d.Category(),
						Snippet:    trimmed,
						Context:    sc.Context(i+1, 3),
					})
					break
				}
			}
		}
	}
	return issues
}
