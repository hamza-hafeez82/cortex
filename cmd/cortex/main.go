package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/hamza-hafeez82/cortex/internal/ai"
	"github.com/hamza-hafeez82/cortex/internal/engine/architecture"
	"github.com/hamza-hafeez82/cortex/internal/engine/recon"
	"github.com/hamza-hafeez82/cortex/internal/engine/security"
	"github.com/hamza-hafeez82/cortex/internal/report"
	"github.com/hamza-hafeez82/cortex/internal/tui"
	"github.com/hamza-hafeez82/cortex/internal/version"
	"github.com/hamza-hafeez82/cortex/internal/walker"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ── Root ──────────────────────────────────────────────────────────────────────

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cortex",
		Short: "A three-stage CLI that audits your codebase for security, architecture, and dependency issues",
		Long: lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF")).Bold(true).Render(`
  ██████╗ ██████╗ ██████╗ ████████╗███████╗██╗  ██╗
 ██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝██╔════╝╚██╗██╔╝
 ██║     ██║   ██║██████╔╝   ██║   █████╗   ╚███╔╝
 ██║     ██║   ██║██╔══██╗   ██║   ██╔══╝   ██╔██╗
 ╚██████╗╚██████╔╝██║  ██║   ██║   ███████╗██╔╝ ██╗
  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝`) + `

  Local-first code security and architecture scanner.
  No data leaves your machine.
`,
		Version:           version.String(),
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	root.AddCommand(
		scanCmd(),
		explainCmd(),
		reportCmd(),
		configCmd(),
	)

	return root
}

// ── cortex scan ───────────────────────────────────────────────────────────────

func scanCmd() *cobra.Command {
	var (
		jsonOutput      bool
		minSeverity     string
		noTUI           bool
		outputFormat    string
	)

	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a repository for security, architecture, and dependency issues",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}

			// JSON flag implies no TUI
			if jsonOutput {
				noTUI = true
			}

			return runScan(target, minSeverity, jsonOutput, outputFormat, noTUI)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON (for CI/CD)")
	cmd.Flags().StringVar(&minSeverity, "severity", "info", "Minimum severity to report (critical, high, medium, low, info)")
	cmd.Flags().BoolVar(&noTUI, "no-tui", false, "Disable interactive TUI, use plain output")
	cmd.Flags().StringVar(&outputFormat, "format", "terminal", "Output format: terminal, json, markdown")

	return cmd
}

func runScan(target, minSeverity string, jsonOutput bool, format string, noTUI bool) error {
	if jsonOutput {
		format = "json"
	}

	// ── Stage 0: Walk the repo ────────────────────────────────────────────
	if !noTUI {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("  Walking repository..."))
	}

	w := walker.New(walker.DefaultOptions())
	repo, err := w.Walk(target)
	if err != nil {
		return fmt.Errorf("failed to walk repository: %w", err)
	}

	if repo.TotalFiles == 0 {
		return fmt.Errorf("no source files found in %s", target)
	}

	updateChan := make(chan tui.StageUpdate, 10)
	var allIssues []detector.Issue

	if noTUI {
		// Plain mode — run stages synchronously with simple progress output
		var err error
		allIssues, err = runPlain(repo, updateChan, noTUI)
		if err != nil {
			return err
		}
	} else {
		// TUI mode — run stages in background, TUI reads updates
		tuiModel := tui.NewModel(target, repo.TotalFiles, updateChan)

		issuesChan := make(chan struct {
			issues []detector.Issue
			err    error
		}, 1)
		go func() {
			issues, err := runStages(repo, updateChan)
			issuesChan <- struct {
				issues []detector.Issue
				err    error
			}{issues, err}
		}()

		p := tea.NewProgram(tuiModel)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		res := <-issuesChan
		if res.err != nil {
			return res.err
		}
		allIssues = res.issues
	}

	// ── Filter by severity ────────────────────────────────────────────────
	filtered := filterBySeverity(allIssues, minSeverity)

	// ── Build and render report ───────────────────────────────────────────
	r := report.NewReport(target, repo.TotalFiles, repo.TotalLines, filtered)

	switch format {
	case "json":
		out, err := r.RenderJSON()
		if err != nil {
			return err
		}
		fmt.Println(out)
	case "markdown":
		fmt.Println(r.RenderMarkdown())
	default:
		fmt.Print(r.RenderTerminal())
	}

	// Exit with code 1 if critical or high issues found (useful for CI)
	for _, issue := range filtered {
		if issue.Severity == detector.SeverityCritical || issue.Severity == detector.SeverityHigh {
			os.Exit(1)
		}
	}

	return nil
}

// runStages executes all three stages and sends updates to the TUI channel.
func runStages(repo *walker.RepoMap, updateChan chan tui.StageUpdate) ([]detector.Issue, error) {
	var allIssues []detector.Issue
	ctx := &detector.ScanContext{Repo: repo}

	// Stage 1: Recon
	reconRunner := recon.NewRunner()
	if _, err := reconRunner.Run(repo); err != nil {
		updateChan <- tui.StageUpdate{Err: err}
		return nil, err
	}
	updateChan <- tui.StageUpdate{Stage: tui.StageRecon, Issues: nil}

	// Stage 2: Security
	secRunner := security.NewRunner()
	secIssues := secRunner.Run(ctx)
	allIssues = append(allIssues, secIssues...)
	updateChan <- tui.StageUpdate{Stage: tui.StageSecurity, Issues: secIssues}

	// Stage 3: Architecture
	archRunner := architecture.NewRunner()
	archIssues := archRunner.Run(ctx)
	allIssues = append(allIssues, archIssues...)
	updateChan <- tui.StageUpdate{Stage: tui.StageArchitecture, Issues: archIssues, Done: true}

	return allIssues, nil
}

func runPlain(repo *walker.RepoMap, _ chan tui.StageUpdate, _ bool) ([]detector.Issue, error) {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	tick := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Bold(true)
	ctx := &detector.ScanContext{Repo: repo}

	fmt.Print(dim.Render("  Stage 1  Reconnaissance..."))
	reconRunner := recon.NewRunner()
	if _, err := reconRunner.Run(repo); err != nil {
		return nil, err
	}
	fmt.Println("  " + tick.Render("done"))

	fmt.Print(dim.Render("  Stage 2  Security scanning..."))
	secRunner := security.NewRunner()
	secIssues := secRunner.Run(ctx)
	fmt.Printf("  %s  %s\n", tick.Render("done"), dim.Render(fmt.Sprintf("%d issues", len(secIssues))))

	fmt.Print(dim.Render("  Stage 3  Architecture analysis..."))
	archRunner := architecture.NewRunner()
	archIssues := archRunner.Run(ctx)
	fmt.Printf("  %s  %s\n\n", tick.Render("done"), dim.Render(fmt.Sprintf("%d issues", len(archIssues))))

	return append(secIssues, archIssues...), nil
}

// ── cortex explain ────────────────────────────────────────────────────────────

func explainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <CODE>",
		Short: "Get an AI-powered explanation of any issue code (e.g. CX-SEC-001)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := strings.ToUpper(args[0])
			return runExplain(code)
		},
	}
}

func runExplain(code string) error {
	cfg, err := ai.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider, err := ai.NewProvider(cfg)
	if err != nil {
		return err
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D4FF"))

	fmt.Println()
	fmt.Printf("  %s  %s\n\n",
		header.Render(code),
		dim.Render("asking "+provider.Name()+"..."),
	)

	// Build a synthetic issue for the explain prompt
	issue := detector.Issue{
		Code:    code,
		Title:   issueTitleFromCode(code),
		Message: "User requested explanation for " + code,
		File:    "—",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	explanation, err := provider.Explain(ctx, issue, "")
	if err != nil {
		return fmt.Errorf("AI explanation failed: %w", err)
	}

	fmt.Println(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Padding(1, 2).
		Width(72).
		Render(explanation))
	fmt.Println()

	return nil
}

func issueTitleFromCode(code string) string {
	titles := map[string]string{
		"CX-SEC-001": "Hardcoded Secrets",
		"CX-SEC-002": "SQL Injection",
		"CX-SEC-003": "Command Injection",
		"CX-SEC-004": "Path Traversal",
		"CX-SEC-005": "Insecure Random",
		"CX-SEC-006": "JWT Misconfiguration",
		"CX-SEC-007": "CORS Misconfiguration",
		"CX-SEC-008": "Sensitive Data in Logs",
		"CX-DEP-001": "Vulnerable Dependency",
		"CX-ARCH-001": "God File",
		"CX-ARCH-002": "Circular Dependency",
		"CX-ARCH-003": "Deep Nesting",
		"CX-ARCH-004": "Missing Error Handling",
		"CX-ARCH-005": "Magic Numbers",
		"CX-ARCH-006": "Dead Code",
		"CX-ARCH-007": "Missing Tests",
		"CX-ARCH-008": "Inconsistent Naming",
	}
	if t, ok := titles[code]; ok {
		return t
	}
	return code
}

// ── cortex report ─────────────────────────────────────────────────────────────

func reportCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "report [path]",
		Short: "Re-run scan and export report in a specific format",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}
			return runScan(target, "info", format == "json", format, true)
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "Output format: terminal, json, markdown")
	return cmd
}

// ── cortex config ─────────────────────────────────────────────────────────────

func configCmd() *cobra.Command {
	var (
		provider string
		apiKey   string
		model    string
		baseURL  string
	)

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure the AI backend for cortex explain",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(provider, apiKey, model, baseURL)
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "AI provider: ollama, openai, anthropic, openai-compat")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for cloud providers")
	cmd.Flags().StringVar(&model, "model", "", "Model name (e.g. llama3, gpt-4o-mini, claude-3-5-haiku-20241022)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL for Ollama or openai-compat providers")

	return cmd
}

func runConfig(providerStr, apiKey, model, baseURL string) error {
	cfg, err := ai.LoadConfig()
	if err != nil {
		cfg = ai.DefaultConfig()
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	tick := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Bold(true)
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D4FF"))

	fmt.Println()

	if providerStr != "" {
		cfg.Provider = ai.ProviderType(providerStr)
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if model != "" {
		cfg.Model = model
	}
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	// If no flags passed, show current config
	if providerStr == "" && apiKey == "" && model == "" && baseURL == "" {
		fmt.Println(header.Render("  Current configuration:"))
		fmt.Println()
		fmt.Printf("  Provider  %s\n", dim.Render(string(cfg.Provider)))
		fmt.Printf("  Model     %s\n", dim.Render(cfg.Model))
		if cfg.APIKey != "" {
			fmt.Printf("  API Key   %s\n", dim.Render(cfg.APIKey[:min(8, len(cfg.APIKey))]+"..."))
		}
		if cfg.BaseURL != "" {
			fmt.Printf("  Base URL  %s\n", dim.Render(cfg.BaseURL))
		}
		fmt.Println()
		fmt.Println(dim.Render("  Use cortex config --provider <name> to change providers"))
		fmt.Println()
		return nil
	}

	if err := ai.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("  %s  Config saved to %s\n",
		tick.Render("✓"),
		dim.Render(ai.ConfigPath()),
	)
	fmt.Printf("  %s  Provider: %s\n",
		tick.Render("✓"),
		header.Render(string(cfg.Provider)),
	)

	// Verify connectivity
	provider, err := ai.NewProvider(cfg)
	if err == nil && provider.IsAvailable() {
		fmt.Printf("  %s  %s is reachable\n\n", tick.Render("✓"), provider.Name())
	} else {
		fmt.Println(dim.Render("\n  Note: provider connectivity could not be verified — check your key or connection"))
	}

	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func filterBySeverity(issues []detector.Issue, minSev string) []detector.Issue {
	order := map[string]int{
		detector.SeverityCritical: 0,
		detector.SeverityHigh:     1,
		detector.SeverityMedium:   2,
		detector.SeverityLow:      3,
		detector.SeverityInfo:     4,
	}

	threshold, ok := order[minSev]
	if !ok {
		threshold = 4
	}

	var filtered []detector.Issue
	for _, issue := range issues {
		if v, ok := order[issue.Severity]; ok && v <= threshold {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
