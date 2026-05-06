package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type CommandInjectionDetector struct{}

func (d *CommandInjectionDetector) ID() string       { return "CX-SEC-003" }
func (d *CommandInjectionDetector) Name() string     { return "Command Injection" }
func (d *CommandInjectionDetector) Category() string { return detector.CategorySecurity }
func (d *CommandInjectionDetector) Severity() string { return detector.SeverityHigh }

var execFunctions = []string{
	"exec(", "execSync(", "execFile(", "spawn(", "spawnSync(",
	"os.system(", "os.popen(", "subprocess.call(", "subprocess.run(",
	"subprocess.Popen(", "shell=True",
	"exec.Command(", "syscall.Exec(",
	"Runtime.exec(", "ProcessBuilder(",
	`sh -c`, "`",
}

var userInputPatterns = []string{
	"req.", "request.", "params.", "query.", "body.", "args.",
	"input(", "stdin", "argv", "sys.argv",
	"os.Args", "flag.", "user_input", "user_data",
	"+", "format(", "sprintf", "Sprintf",
}

var safeExecPatterns = []string{"hardcoded", "const ", "\"ls\"", "\"pwd\"", "\"echo\""}

func (d *CommandInjectionDetector) Run(ctx *detector.ScanContext) []detector.Issue {
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

			hasExec := false
			for _, fn := range execFunctions {
				if strings.Contains(trimmed, fn) {
					hasExec = true
					break
				}
			}
			if !hasExec {
				continue
			}

			hasUserInput := false
			for _, pat := range userInputPatterns {
				if strings.Contains(trimmed, pat) {
					hasUserInput = true
					break
				}
			}
			if !hasUserInput {
				continue
			}

			isSafe := false
			for _, safe := range safeExecPatterns {
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
				Message:    "User input passed to shell execution — sanitize input and avoid shell=True or dynamic command construction",
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
