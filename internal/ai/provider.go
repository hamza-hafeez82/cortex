package ai

import (
	"context"
	"fmt"

	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// ProviderType identifies which AI backend to use.
type ProviderType string

const (
	ProviderOllama    ProviderType = "ollama"
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderCompat    ProviderType = "openai-compat"
)

// Config holds the AI backend configuration stored in ~/.cortex/config.yaml.
type Config struct {
	Provider ProviderType `yaml:"provider"`
	Model    string       `yaml:"model"`
	APIKey   string       `yaml:"api_key"`
	BaseURL  string       `yaml:"base_url"` // for openai-compat
	Timeout  int          `yaml:"timeout"`  // seconds, default 30
}

// DefaultConfig returns sensible defaults — Ollama with llama3.
func DefaultConfig() Config {
	return Config{
		Provider: ProviderOllama,
		Model:    "llama3",
		BaseURL:  "http://localhost:11434",
		Timeout:  30,
	}
}

// Provider is the interface every AI backend implements.
type Provider interface {
	// Name returns the provider display name.
	Name() string

	// Explain sends the issue and its code context to the AI and returns
	// a detailed explanation. The context.Context controls cancellation/timeout.
	Explain(ctx context.Context, issue detector.Issue, codeContext string) (string, error)

	// IsAvailable performs a lightweight connectivity check.
	IsAvailable() bool
}

// ExplainPrompt builds the prompt sent to any AI provider for issue explanation.
func ExplainPrompt(issue detector.Issue, codeContext string) string {
	return fmt.Sprintf(`You are a senior security engineer reviewing code. Explain the following issue found by Cortex, a static analysis tool.

Issue Code: %s
Issue Title: %s  
Severity: %s
File: %s (line %d)
Message: %s

Relevant code:
%s

Provide:
1. A clear explanation of WHY this is a problem
2. The specific security or quality risk it introduces
3. A concrete fix with corrected code example
4. Any relevant best practices

Be direct and practical. No unnecessary preamble.`,
		issue.Code,
		issue.Title,
		issue.Severity,
		issue.File,
		issue.Line,
		issue.Message,
		codeContext,
	)
}
