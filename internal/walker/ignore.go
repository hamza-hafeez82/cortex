package walker

import (
	"path/filepath"
	"strings"
)

// ignoredDirs are directory names that should never be scanned.
// Matched against each path component individually.
var ignoredDirs = map[string]bool{
	".git":             true,
	".svn":             true,
	".hg":              true,
	"node_modules":     true,
	"vendor":           true,
	"dist":             true,
	"build":            true,
	"out":              true,
	".next":            true,
	".nuxt":            true,
	".cache":           true,
	".parcel-cache":    true,
	"__pycache__":      true,
	".pytest_cache":    true,
	".mypy_cache":      true,
	"venv":             true,
	".venv":            true,
	"env":              true,
	".env":             true,
	"site-packages":    true,
	"target":           true, // Rust/Java build output
	".gradle":          true,
	".idea":            true,
	".vscode":          true,
	"coverage":         true,
	".nyc_output":      true,
	"htmlcov":          true,
	"storybook-static": true,
	"public":           true, // typically compiled assets
	"static":           true,
}

// ignoredExtensions are file extensions that should never be scanned.
var ignoredExtensions = map[string]bool{
	// Compiled / binary
	".exe":   true,
	".bin":   true,
	".dll":   true,
	".so":    true,
	".dylib": true,
	".a":     true,
	".o":     true,
	".obj":   true,
	".class": true,
	".pyc":   true,
	".pyo":   true,
	".wasm":  true,

	// Archives
	".zip": true,
	".tar": true,
	".gz":  true,
	".tgz": true,
	".bz2": true,
	".xz":  true,
	".7z":  true,
	".rar": true,
	".jar": true,
	".war": true,

	// Media
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".bmp":   true,
	".ico":   true,
	".svg":   true,
	".webp":  true,
	".mp4":   true,
	".mp3":   true,
	".wav":   true,
	".ogg":   true,
	".avi":   true,
	".mov":   true,
	".webm":  true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
	".otf":   true,

	// Documents / data blobs
	".pdf":    true,
	".doc":    true,
	".docx":   true,
	".xls":    true,
	".xlsx":   true,
	".pptx":   true,
	".sqlite": true,
	".db":     true,

	// Lockfiles (useful for dep analysis but not line scanning)
	".lock": true,
}

// ignoredFilenames are exact filenames that should be skipped.
var ignoredFilenames = map[string]bool{
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"Cargo.lock":        true,
	"poetry.lock":       true,
	"composer.lock":     true,
	".DS_Store":         true,
	"Thumbs.db":         true,
}

// ShouldIgnorePath returns true if the given absolute path should be
// excluded from scanning. It checks each path component against
// ignoredDirs and checks the file extension and filename.
func ShouldIgnorePath(path string) bool {
	// Normalize to forward slashes for consistent splitting
	normalized := filepath.ToSlash(path)
	parts := strings.Split(normalized, "/")

	for _, part := range parts {
		if ignoredDirs[part] {
			return true
		}
	}

	return false
}

// ShouldIgnoreFile returns true if the file itself (not its directory)
// should be excluded based on its name or extension.
func ShouldIgnoreFile(name, ext string) bool {
	if ignoredFilenames[name] {
		return true
	}
	if ignoredExtensions[strings.ToLower(ext)] {
		return true
	}
	return false
}
