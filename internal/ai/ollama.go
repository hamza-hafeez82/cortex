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

// OllamaProvider talks to a locally running Ollama instance.
type OllamaProvider struct {
	cfg    Config
	client *http.Client
}

// NewOllamaProvider creates an Ollama provider from config.
func NewOllamaProvider(cfg Config) *OllamaProvider {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *OllamaProvider) Name() string { return "Ollama (" + p.cfg.Model + ")" }

func (p *OllamaProvider) IsAvailable() bool {
	resp, err := p.client.Get(p.cfg.BaseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *OllamaProvider) Explain(ctx context.Context, issue detector.Issue, codeContext string) (string, error) {
	prompt := ExplainPrompt(issue, codeContext)

	body, _ := json.Marshal(map[string]interface{}{
		"model":  p.cfg.Model,
		"prompt": prompt,
		"stream": false,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building Ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading Ollama response: %w", err)
	}

	var result struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing Ollama response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", result.Error)
	}

	return result.Response, nil
}
