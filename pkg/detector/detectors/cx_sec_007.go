package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type CORSMisconfigDetector struct{}

func (d *CORSMisconfigDetector) ID() string       { return "CX-SEC-007" }
func (d *CORSMisconfigDetector) Name() string     { return "CORS Misconfiguration" }
func (d *CORSMisconfigDetector) Category() string { return detector.CategorySecurity }
func (d *CORSMisconfigDetector) Severity() string { return detector.SeverityMedium }

var corsDangerousPatterns = []struct {
	pattern string
	message string
}{
	{`"*"`, "Wildcard CORS origin detected"},
	{`'*'`, "Wildcard CORS origin detected"},
	{"origin: '*'", "Wildcard CORS origin allows any domain to make requests"},
	{`origin: "*"`, "Wildcard CORS origin allows any domain to make requests"},
	{"AllowAllOrigins: true", "All origins allowed — scope CORS to trusted domains only"},
	{"allow_origins=[\"*\"]", "Wildcard CORS origin in FastAPI/Starlette — restrict to known domains"},
	{"allow_origins=['*']", "Wildcard CORS origin in FastAPI/Starlette — restrict to known domains"},
	{"Access-Control-Allow-Origin: *", "Wildcard CORS header set directly — restrict to trusted origins"},
	{`Header("Access-Control-Allow-Origin", "*")`, "Wildcard CORS header — restrict to trusted origins"},
}

var corsCredentialKeywords = []string{
	"credentials: true", "AllowCredentials: true",
	"allow_credentials=True", "withCredentials",
}

func (d *CORSMisconfigDetector) Run(ctx *detector.ScanContext) []detector.Issue {
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

			for _, pat := range corsDangerousPatterns {
				if !strings.Contains(trimmed, pat.pattern) {
					continue
				}

				// Check if credentials are also enabled — makes it critical
				severity := d.Severity()
				contextLines := sc.Context(i+1, 5)
				contextStr := strings.Join(contextLines, " ")
				for _, cred := range corsCredentialKeywords {
					if strings.Contains(contextStr, cred) {
						severity = detector.SeverityHigh
						break
					}
				}

				issues = append(issues, detector.Issue{
					Code:       d.ID(),
					Title:      d.Name(),
					Message:    pat.message,
					File:       f.Path,
					Line:       i + 1,
					Severity:   severity,
					Confidence: detector.ConfidenceHigh,
					Category:   d.Category(),
					Snippet:    trimmed,
					Context:    sc.Context(i+1, 3),
				})
				break
			}
		}
	}
	return issues
}
