package parser_test

import (
	"testing"

	"github.com/hamza-hafeez82/cortex/internal/parser"
)

func TestDetectLanguageByExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected string
	}{
		{"main.go", ".go", parser.LangGo},
		{"app.js", ".js", parser.LangJavaScript},
		{"app.mjs", ".mjs", parser.LangJavaScript},
		{"component.jsx", ".jsx", parser.LangJavaScript},
		{"server.ts", ".ts", parser.LangTypeScript},
		{"component.tsx", ".tsx", parser.LangTypeScript},
		{"main.py", ".py", parser.LangPython},
		{"lib.rs", ".rs", parser.LangRust},
		{"Main.java", ".java", parser.LangJava},
		{"App.kt", ".kt", parser.LangKotlin},
		{"ViewController.swift", ".swift", parser.LangSwift},
		{"Program.cs", ".cs", parser.LangCSharp},
		{"main.c", ".c", parser.LangC},
		{"main.cpp", ".cpp", parser.LangCPP},
		{"app.rb", ".rb", parser.LangRuby},
		{"index.php", ".php", parser.LangPHP},
		{"deploy.sh", ".sh", parser.LangShell},
		{"config.yml", ".yml", parser.LangYAML},
		{"config.yaml", ".yaml", parser.LangYAML},
		{"package.json", ".json", parser.LangJSON},
		{"Cargo.toml", ".toml", parser.LangTOML},
		{"README.md", ".md", parser.LangMarkdown},
		{"query.sql", ".sql", parser.LangSQL},
		{"index.html", ".html", parser.LangHTML},
		{"styles.css", ".css", parser.LangCSS},
		{"styles.scss", ".scss", parser.LangCSS},
		{"Main.scala", ".scala", parser.LangScala},
		{"app.ex", ".ex", parser.LangElixir},
		{"App.vue", ".vue", parser.LangVue},
		{"App.svelte", ".svelte", parser.LangSvelte},
		{"main.tf", ".tf", parser.LangTerraform},
		{"service.proto", ".proto", parser.LangProto},
		{"unknown.xyz", ".xyz", parser.LangUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.DetectLanguage(tt.name, tt.ext)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q, %q) = %q, want %q",
					tt.name, tt.ext, got, tt.expected)
			}
		})
	}
}

func TestDetectLanguageByFilename(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Dockerfile", parser.LangDockerfile},
		{"Dockerfile.dev", parser.LangDockerfile},
		{"dockerfile", parser.LangDockerfile},
		{"Makefile", parser.LangShell},
		{"Rakefile", parser.LangRuby},
		{"Gemfile", parser.LangRuby},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.DetectLanguage(tt.name, "")
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q, %q) = %q, want %q",
					tt.name, "", got, tt.expected)
			}
		})
	}
}

func TestIsSourceLanguage(t *testing.T) {
	sourceLangs := []string{
		parser.LangGo, parser.LangJavaScript, parser.LangTypeScript,
		parser.LangPython, parser.LangRust, parser.LangJava,
	}
	for _, lang := range sourceLangs {
		if !parser.IsSourceLanguage(lang) {
			t.Errorf("IsSourceLanguage(%q) = false, want true", lang)
		}
	}

	nonSourceLangs := []string{
		parser.LangYAML, parser.LangJSON, parser.LangMarkdown,
		parser.LangDockerfile, parser.LangUnknown,
	}
	for _, lang := range nonSourceLangs {
		if parser.IsSourceLanguage(lang) {
			t.Errorf("IsSourceLanguage(%q) = true, want false", lang)
		}
	}
}

func TestIsConfigLanguage(t *testing.T) {
	configLangs := []string{
		parser.LangYAML, parser.LangTOML, parser.LangJSON,
		parser.LangDockerfile, parser.LangTerraform,
	}
	for _, lang := range configLangs {
		if !parser.IsConfigLanguage(lang) {
			t.Errorf("IsConfigLanguage(%q) = false, want true", lang)
		}
	}
}
