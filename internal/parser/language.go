package parser

import "strings"

// Language constants used throughout the codebase.
const (
	LangGo         = "Go"
	LangJavaScript = "JavaScript"
	LangTypeScript = "TypeScript"
	LangPython     = "Python"
	LangRust       = "Rust"
	LangJava       = "Java"
	LangKotlin     = "Kotlin"
	LangSwift      = "Swift"
	LangCSharp     = "C#"
	LangCPP        = "C++"
	LangC          = "C"
	LangRuby       = "Ruby"
	LangPHP        = "PHP"
	LangShell      = "Shell"
	LangYAML       = "YAML"
	LangJSON       = "JSON"
	LangTOML       = "TOML"
	LangMarkdown   = "Markdown"
	LangDockerfile = "Dockerfile"
	LangSQL        = "SQL"
	LangHTML       = "HTML"
	LangCSS        = "CSS"
	LangScala      = "Scala"
	LangElixir     = "Elixir"
	LangHaskell    = "Haskell"
	LangLua        = "Lua"
	LangVue        = "Vue"
	LangSvelte     = "Svelte"
	LangTerraform  = "Terraform"
	LangProto      = "Protobuf"
	LangUnknown    = "Unknown"
)

// extToLanguage maps lowercase file extensions to language names.
var extToLanguage = map[string]string{
	// Go
	".go": LangGo,

	// JavaScript / TypeScript
	".js":  LangJavaScript,
	".mjs": LangJavaScript,
	".cjs": LangJavaScript,
	".jsx": LangJavaScript,
	".ts":  LangTypeScript,
	".tsx": LangTypeScript,
	".mts": LangTypeScript,
	".cts": LangTypeScript,

	// Python
	".py":  LangPython,
	".pyw": LangPython,
	".pyi": LangPython,

	// Rust
	".rs": LangRust,

	// Java
	".java": LangJava,

	// Kotlin
	".kt":  LangKotlin,
	".kts": LangKotlin,

	// Swift
	".swift": LangSwift,

	// C#
	".cs": LangCSharp,

	// C / C++
	".c":   LangC,
	".h":   LangC,
	".cpp": LangCPP,
	".cc":  LangCPP,
	".cxx": LangCPP,
	".hpp": LangCPP,
	".hxx": LangCPP,

	// Ruby
	".rb":      LangRuby,
	".rake":    LangRuby,
	".gemspec": LangRuby,

	// PHP
	".php": LangPHP,

	// Shell
	".sh":   LangShell,
	".bash": LangShell,
	".zsh":  LangShell,
	".fish": LangShell,

	// Config / data formats
	".yaml": LangYAML,
	".yml":  LangYAML,
	".json": LangJSON,
	".toml": LangTOML,

	// Markup
	".md":       LangMarkdown,
	".markdown": LangMarkdown,
	".html":     LangHTML,
	".htm":      LangHTML,
	".css":      LangCSS,
	".scss":     LangCSS,
	".sass":     LangCSS,
	".less":     LangCSS,

	// SQL
	".sql": LangSQL,

	// Scala
	".scala": LangScala,
	".sc":    LangScala,

	// Elixir
	".ex":  LangElixir,
	".exs": LangElixir,

	// Haskell
	".hs":  LangHaskell,
	".lhs": LangHaskell,

	// Lua
	".lua": LangLua,

	// Frontend frameworks
	".vue":    LangVue,
	".svelte": LangSvelte,

	// Infrastructure
	".tf":     LangTerraform,
	".tfvars": LangTerraform,

	// Protobuf
	".proto": LangProto,
}

// filenameToLanguage maps exact filenames (no extension) to language names.
// Takes priority over extension detection.
var filenameToLanguage = map[string]string{
	"Dockerfile":          LangDockerfile,
	"dockerfile":          LangDockerfile,
	"Makefile":            LangShell,
	"makefile":            LangShell,
	"GNUmakefile":         LangShell,
	"Rakefile":            LangRuby,
	"Gemfile":             LangRuby,
	"Guardfile":           LangRuby,
	"Fastfile":            LangRuby,
	"Podfile":             LangRuby,
	".bashrc":             LangShell,
	".zshrc":              LangShell,
	".bash_profile":       LangShell,
	".profile":            LangShell,
	"docker-compose.yml":  LangYAML,
	"docker-compose.yaml": LangYAML,
}

// DetectLanguage returns the programming language for a file given its name
// and lowercased extension. Filename-based detection takes priority.
func DetectLanguage(name, ext string) string {
	// Check exact filename first (handles Dockerfile, Makefile, etc.)
	if lang, ok := filenameToLanguage[name]; ok {
		return lang
	}

	// Handle Dockerfile variants (e.g. Dockerfile.dev, Dockerfile.prod)
	if strings.HasPrefix(name, "Dockerfile") || strings.HasPrefix(name, "dockerfile") {
		return LangDockerfile
	}

	// Check extension
	if lang, ok := extToLanguage[strings.ToLower(ext)]; ok {
		return lang
	}

	return LangUnknown
}

// IsSourceLanguage returns true if the language is a primary programming
// language (not config, markup, or data). Used to limit certain detectors
// to code files only.
func IsSourceLanguage(lang string) bool {
	switch lang {
	case LangGo, LangJavaScript, LangTypeScript, LangPython, LangRust,
		LangJava, LangKotlin, LangSwift, LangCSharp, LangCPP, LangC,
		LangRuby, LangPHP, LangScala, LangElixir, LangHaskell, LangLua:
		return true
	}
	return false
}

// IsConfigLanguage returns true if the language is a configuration or
// infrastructure format. Used by the recon stage.
func IsConfigLanguage(lang string) bool {
	switch lang {
	case LangYAML, LangTOML, LangJSON, LangDockerfile, LangTerraform:
		return true
	}
	return false
}
