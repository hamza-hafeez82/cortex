package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type SensitiveDataInLogsDetector struct{}

func (d *SensitiveDataInLogsDetector) ID() string       { return "CX-SEC-008" }
func (d *SensitiveDataInLogsDetector) Name() string     { return "Sensitive Data in Logs" }
func (d *SensitiveDataInLogsDetector) Category() string { return detector.CategorySecurity }
func (d *SensitiveDataInLogsDetector) Severity() string { return detector.SeverityMedium }

var logFunctions = []string{
	"console.log(", "console.error(", "console.warn(", "console.debug(",
	"fmt.Println(", "fmt.Printf(", "fmt.Print(", "log.Println(", "log.Printf(", "log.Print(",
	"log.Fatal(", "log.Fatalf(", "logger.info(", "logger.debug(", "logger.error(", "logger.warn(",
	"print(", "logging.info(", "logging.debug(", "logging.warning(", "logging.error(",
	"System.out.println(", "System.err.println(",
}

var sensitiveDataKeywords = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"private_key", "credit_card", "cvv", "ssn", "social_security",
	"auth", "authorization", "bearer", "jwt", "session",
}

func (d *SensitiveDataInLogsDetector) Run(ctx *detector.ScanContext) []detector.Issue {
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

			hasLogFn := false
			for _, fn := range logFunctions {
				if strings.Contains(trimmed, fn) {
					hasLogFn = true
					break
				}
			}
			if !hasLogFn {
				continue
			}

			lineLower := strings.ToLower(trimmed)
			for _, kw := range sensitiveDataKeywords {
				if strings.Contains(lineLower, kw) {
					issues = append(issues, detector.Issue{
						Code:       d.ID(),
						Title:      d.Name(),
						Message:    "Sensitive field '" + kw + "' may be written to logs — redact sensitive data before logging",
						File:       f.Path,
						Line:       i + 1,
						Severity:   d.Severity(),
						Confidence: detector.ConfidenceMedium,
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
