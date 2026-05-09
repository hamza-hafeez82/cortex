package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigPath returns the path to the Cortex config file.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cortex", "config.yaml")
}

// LoadConfig reads the config from ~/.cortex/config.yaml.
// Returns DefaultConfig if the file doesn't exist.
func LoadConfig() (Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes the config to ~/.cortex/config.yaml.
func SaveConfig(cfg Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

// NewProvider creates the appropriate Provider from config.
// If Ollama is configured and available, it takes priority.
// Falls back to auto-detecting based on available API keys.
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case ProviderOllama, "":
		p := NewOllamaProvider(cfg)
		if p.IsAvailable() {
			return p, nil
		}
		// Auto-fallback if API keys are set
		if cfg.APIKey != "" {
			return NewOpenAIProvider(cfg), nil
		}
		return nil, fmt.Errorf(
			"Ollama is not running at %s\n\n"+
				"Start Ollama:  ollama serve\n"+
				"Pull a model:  ollama pull llama3\n\n"+
				"Or configure an API key:\n"+
				"  cortex config --provider openai --api-key sk-...",
			cfg.BaseURL,
		)

	case ProviderOpenAI:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key not set — run: cortex config --provider openai --api-key sk-...")
		}
		return NewOpenAIProvider(cfg), nil

	case ProviderAnthropic:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("Anthropic API key not set — run: cortex config --provider anthropic --api-key sk-ant-...")
		}
		return NewAnthropicProvider(cfg), nil

	case ProviderCompat:
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("base URL required for openai-compat provider")
		}
		return NewOpenAIProvider(cfg), nil

	default:
		return nil, fmt.Errorf("unknown provider %q — valid options: ollama, openai, anthropic, openai-compat", cfg.Provider)
	}
}
