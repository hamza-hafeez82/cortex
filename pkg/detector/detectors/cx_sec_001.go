package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type HardcodedSecretsDetector struct{}

func (d *HardcodedSecretsDetector) ID() string       { return "CX-SEC-001" }
func (d *HardcodedSecretsDetector) Name() string     { return "Hardcoded Secrets" }
func (d *HardcodedSecretsDetector) Category() string { return detector.CategorySecurity }
func (d *HardcodedSecretsDetector) Severity() string { return detector.SeverityCritical }

var secretKeywords = []string{
	"api_key", "apikey", "api_secret", "secret_key", "secretkey",
	"password", "passwd", "auth_token", "access_token", "private_key",
	"client_secret", "db_password", "aws_secret", "aws_access_key",
	"stripe_key", "stripe_secret", "jwt_secret", "token_secret",
}

var suspiciousValuePrefixes = []string{
	"sk-", "pk-", "ghp_", "gho_", "xox", "AKIA", "eyJ", "-----BEGIN", "AIza", "ya29.",
}

var ignoredValuePatterns = []string{
	"os.getenv", "os.environ", "process.env", "getenv(", "environ.get(",
	"config.", "cfg.", "settings.", "vault.", "<", ">", "example",
	"placeholder", "your_", "todo", `""`, `''`, "nil", "null", "none",
}

func (d *HardcodedSecretsDetector) Run(ctx *detector.ScanContext) []detector.Issue {
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
			lineLower := strings.ToLower(trimmed)

			hasKeyword := false
			for _, kw := range secretKeywords {
				if strings.Contains(lineLower, kw) {
					hasKeyword = true
					break
				}
			}
			if !hasKeyword {
				continue
			}

			hasAssignment := strings.Contains(trimmed, "=") || strings.Contains(trimmed, ": ")
			if !hasAssignment {
				continue
			}

			isIgnored := false
			for _, ig := range ignoredValuePatterns {
				if strings.Contains(lineLower, ig) {
					isIgnored = true
					break
				}
			}
			if isIgnored {
				continue
			}

			if !strings.Contains(trimmed, `"`) && !strings.Contains(trimmed, `'`) {
				continue
			}

			confidence := detector.ConfidenceMedium
			for _, pat := range suspiciousValuePrefixes {
				if strings.Contains(trimmed, pat) {
					confidence = detector.ConfidenceHigh
					break
				}
			}

			issues = append(issues, detector.Issue{
				Code:       d.ID(),
				Title:      d.Name(),
				Message:    "Potential hardcoded secret — use environment variables or a secrets manager",
				File:       f.Path,
				Line:       i + 1,
				Severity:   d.Severity(),
				Confidence: confidence,
				Category:   d.Category(),
				Snippet:    trimmed,
				Context:    sc.Context(i+1, 3),
			})
		}
	}
	return issues
}
