package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// AnthropicProvider calls the Anthropic Messages API.
type AnthropicProvider struct {
	cfg    Config
	client *http.Client
}

// NewAnthropicProvider creates an Anthropic provider from config.
func NewAnthropicProvider(cfg Config) *AnthropicProvider {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if cfg.Model == "" {
		cfg.Model = "claude-3-5-haiku-20241022"
	}
	return &AnthropicProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *AnthropicProvider) Name() string { return "Anthropic (" + p.cfg.Model + ")" }

func (p *AnthropicProvider) IsAvailable() bool {
	return p.cfg.APIKey != ""
}

func (p *AnthropicProvider) Explain(ctx context.Context, issue detector.Issue, codeContext string) (string, error) {
	prompt := ExplainPrompt(issue, codeContext)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":      p.cfg.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("building Anthropic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Anthropic: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading Anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic returned status %d: %s", resp.StatusCode, string(data))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing Anthropic response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("Anthropic error: %s", result.Error.Message)
	}
	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("Anthropic returned no text content")
}
