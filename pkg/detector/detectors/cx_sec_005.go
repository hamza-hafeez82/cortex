package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type InsecureRandomDetector struct{}

func (d *InsecureRandomDetector) ID() string       { return "CX-SEC-005" }
func (d *InsecureRandomDetector) Name() string     { return "Insecure Random" }
func (d *InsecureRandomDetector) Category() string { return detector.CategorySecurity }
func (d *InsecureRandomDetector) Severity() string { return detector.SeverityMedium }

var insecureRandomFunctions = []string{
	"Math.random()", "math.random()",
	"rand.Intn(", "rand.Int(", "rand.Float",
	"random.random(", "random.randint(", "random.choice(",
	"rand()", "mt_rand(",
}

var securityContextKeywords = []string{
	"token", "secret", "key", "password", "salt", "nonce",
	"csrf", "session", "auth", "otp", "pin", "uuid", "id",
}

var safeRandomPatterns = []string{
	"crypto/rand", "crypto.randomBytes", "crypto.getRandomValues",
	"secrets.", "os.urandom", "SecureRandom",
}

func (d *InsecureRandomDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}
		sc := parser.NewScanner(f.Path, f.Lines)

		// First check if file uses safe random — if so, skip the whole file
		fileContent := strings.Join(f.Lines, "\n")
		for _, safe := range safeRandomPatterns {
			if strings.Contains(fileContent, safe) {
				goto nextFile
			}
		}

		for i, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			if parser.IsComment(trimmed, f.Language) {
				continue
			}

			hasInsecureRand := false
			for _, fn := range insecureRandomFunctions {
				if strings.Contains(trimmed, fn) {
					hasInsecureRand = true
					break
				}
			}
			if !hasInsecureRand {
				continue
			}

			// Check surrounding lines (±5) for security context keywords
			contextLines := sc.Context(i+1, 5)
			contextStr := strings.ToLower(strings.Join(contextLines, " "))

			hasSecurityContext := false
			for _, kw := range securityContextKeywords {
				if strings.Contains(contextStr, kw) {
					hasSecurityContext = true
					break
				}
			}
			if !hasSecurityContext {
				continue
			}

			issues = append(issues, detector.Issue{
				Code:       d.ID(),
				Title:      d.Name(),
				Message:    "Insecure random number generator used in security-sensitive context — use crypto/rand or equivalent",
				File:       f.Path,
				Line:       i + 1,
				Severity:   d.Severity(),
				Confidence: detector.ConfidenceMedium,
				Category:   d.Category(),
				Snippet:    trimmed,
				Context:    sc.Context(i+1, 3),
			})
		}
	nextFile:
	}
	return issues
}
