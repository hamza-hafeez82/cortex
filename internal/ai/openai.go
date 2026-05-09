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

// OpenAIProvider works with OpenAI and any OpenAI-compatible API.
type OpenAIProvider struct {
	cfg    Config
	client *http.Client
}

// NewOpenAIProvider creates an OpenAI provider from config.
func NewOpenAIProvider(cfg Config) *OpenAIProvider {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *OpenAIProvider) Name() string { return "OpenAI (" + p.cfg.Model + ")" }

func (p *OpenAIProvider) IsAvailable() bool {
	return p.cfg.APIKey != ""
}

func (p *OpenAIProvider) Explain(ctx context.Context, issue detector.Issue, codeContext string) (string, error) {
	prompt := ExplainPrompt(issue, codeContext)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": p.cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 1024,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("building OpenAI request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling OpenAI: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading OpenAI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI returned status %d: %s", resp.StatusCode, string(data))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing OpenAI response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("OpenAI error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("OpenAI returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}
