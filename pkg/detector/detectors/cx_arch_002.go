package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// CircularDepsDetector finds circular import relationships between packages.
type CircularDepsDetector struct{}

func (d *CircularDepsDetector) ID() string       { return "CX-ARCH-002" }
func (d *CircularDepsDetector) Name() string     { return "Circular Dependency" }
func (d *CircularDepsDetector) Category() string { return detector.CategoryArchitecture }
func (d *CircularDepsDetector) Severity() string { return detector.SeverityHigh }

func (d *CircularDepsDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	var issues []detector.Issue

	// Build import graph: package → set of imported packages
	importGraph := buildImportGraph(ctx)

	// Detect cycles using DFS
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	reported := make(map[string]bool)

	for pkg := range importGraph {
		if !visited[pkg] {
			cycle := detectCycle(pkg, importGraph, visited, inStack, []string{})
			if cycle != nil {
				key := strings.Join(cycle, "->")
				if !reported[key] {
					reported[key] = true
					// Find a representative file for the first package in the cycle
					file := findPackageFile(ctx, cycle[0])
					issues = append(issues, detector.Issue{
						Code:       d.ID(),
						Title:      d.Name(),
						Message:    "Circular import detected: " + strings.Join(cycle, " → "),
						File:       file,
						Line:       0,
						Severity:   d.Severity(),
						Confidence: detector.ConfidenceHigh,
						Category:   d.Category(),
						Snippet:    strings.Join(cycle, " → "),
					})
				}
			}
		}
	}

	return issues
}

// buildImportGraph constructs a map of package path → imported package paths
// by scanning import statements in Go, Python, and JS/TS files.
func buildImportGraph(ctx *detector.ScanContext) map[string][]string {
	graph := make(map[string][]string)

	for _, f := range ctx.Repo.Files {
		if !parser.IsSourceLanguage(f.Language) || len(f.Lines) == 0 {
			continue
		}

		pkg := packageFromPath(f.Path, f.Language)
		if pkg == "" {
			continue
		}

		imports := extractImports(f.Lines, f.Language, ctx.Repo.Root)
		for _, imp := range imports {
			// Only track internal imports (same repo)
			if isInternalImport(imp, ctx.Repo.Root) {
				graph[pkg] = append(graph[pkg], imp)
			}
		}
	}

	return graph
}

// packageFromPath derives a logical package name from a file path.
func packageFromPath(path, lang string) string {
	// Use the directory as the package identifier
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// extractImports pulls import paths from file lines based on language.
func extractImports(lines []string, lang, root string) []string {
	var imports []string
	inImportBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch lang {
		case parser.LangGo:
			if trimmed == "import (" {
				inImportBlock = true
				continue
			}
			if inImportBlock && trimmed == ")" {
				inImportBlock = false
				continue
			}
			if inImportBlock || strings.HasPrefix(trimmed, "import ") {
				imp := extractGoImport(trimmed)
				if imp != "" {
					imports = append(imports, imp)
				}
			}

		case parser.LangPython:
			if strings.HasPrefix(trimmed, "from ") || strings.HasPrefix(trimmed, "import ") {
				imp := extractPythonImport(trimmed)
				if imp != "" {
					imports = append(imports, imp)
				}
			}

		case parser.LangJavaScript, parser.LangTypeScript:
			if strings.Contains(trimmed, "from '") || strings.Contains(trimmed, `from "`) ||
				strings.Contains(trimmed, "require('") || strings.Contains(trimmed, `require("`) {
				imp := extractJSImport(trimmed)
				if imp != "" {
					imports = append(imports, imp)
				}
			}
		}
	}

	return imports
}

func extractGoImport(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "import ")
	line = strings.Trim(line, `"`)
	// Strip alias
	if idx := strings.Index(line, `"`); idx >= 0 {
		end := strings.Index(line[idx+1:], `"`)
		if end >= 0 {
			return line[idx+1 : idx+1+end]
		}
	}
	return strings.Trim(line, `" `)
}

func extractPythonImport(line string) string {
	if strings.HasPrefix(line, "from .") || strings.HasPrefix(line, "from ..") {
		// Relative import
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	if strings.HasPrefix(line, "from ") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func extractJSImport(line string) string {
	for _, q := range []string{"'", `"`} {
		if idx := strings.LastIndex(line, q); idx > 0 {
			start := strings.LastIndex(line[:idx], q)
			if start >= 0 && start < idx {
				path := line[start+1 : idx]
				if strings.HasPrefix(path, ".") {
					return path
				}
			}
		}
	}
	return ""
}

func isInternalImport(imp, root string) bool {
	return strings.HasPrefix(imp, ".") ||
		strings.Contains(imp, root) ||
		(!strings.Contains(imp, ".") && !strings.HasPrefix(imp, "std"))
}

func findPackageFile(ctx *detector.ScanContext, pkg string) string {
	for _, f := range ctx.Repo.Files {
		if strings.HasPrefix(f.Path, pkg) {
			return f.Path
		}
	}
	return pkg
}

// detectCycle runs DFS to find a cycle in the import graph.
// Returns the cycle path if found, nil otherwise.
func detectCycle(node string, graph map[string][]string, visited, inStack map[string]bool, path []string) []string {
	visited[node] = true
	inStack[node] = true
	path = append(path, node)

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if cycle := detectCycle(neighbor, graph, visited, inStack, path); cycle != nil {
				return cycle
			}
		} else if inStack[neighbor] {
			// Found cycle — return the cycle portion of the path
			for i, p := range path {
				if p == neighbor {
					return append(path[i:], neighbor)
				}
			}
		}
	}

	inStack[node] = false
	return nil
}
