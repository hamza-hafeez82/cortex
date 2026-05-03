package recon

import (
	"encoding/json"
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/walker"
)

// ParseDependencies scans all manifest files in the repo and returns
// all dependencies found across ecosystems.
func ParseDependencies(repo *walker.RepoMap) []Dependency {
	var deps []Dependency

	for _, f := range repo.Files {
		switch f.Name {
		case "package.json":
			deps = append(deps, parsePackageJSON(f)...)
		case "go.mod":
			deps = append(deps, parseGoMod(f)...)
		case "requirements.txt":
			deps = append(deps, parseRequirementsTxt(f)...)
		case "Cargo.toml":
			deps = append(deps, parseCargoToml(f)...)
		case "pyproject.toml":
			deps = append(deps, parsePyprojectToml(f)...)
		case "pom.xml":
			deps = append(deps, parsePomXML(f)...)
		}
	}

	return deps
}

// parsePackageJSON parses npm dependencies from package.json.
func parsePackageJSON(f *walker.FileNode) []Dependency {
	if len(f.Lines) == 0 {
		return nil
	}

	content := strings.Join(f.Lines, "\n")

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil
	}

	var deps []Dependency
	for name, version := range pkg.Dependencies {
		deps = append(deps, Dependency{
			Name:      name,
			Version:   version,
			Ecosystem: "npm",
			Dev:       false,
			Source:    f.Path,
		})
	}
	for name, version := range pkg.DevDependencies {
		deps = append(deps, Dependency{
			Name:      name,
			Version:   version,
			Ecosystem: "npm",
			Dev:       true,
			Source:    f.Path,
		})
	}

	return deps
}

// parseGoMod parses Go module dependencies from go.mod.
func parseGoMod(f *walker.FileNode) []Dependency {
	var deps []Dependency
	inRequire := false

	for _, line := range f.Lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "require (" {
			inRequire = true
			continue
		}
		if inRequire && trimmed == ")" {
			inRequire = false
			continue
		}

		// Single-line require: require github.com/foo/bar v1.2.3
		if strings.HasPrefix(trimmed, "require ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				deps = append(deps, Dependency{
					Name:      parts[1],
					Version:   parts[2],
					Ecosystem: "go",
					Source:    f.Path,
				})
			}
			continue
		}

		// Inside require block
		if inRequire && trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			// Strip inline comment
			if idx := strings.Index(trimmed, "//"); idx >= 0 {
				trimmed = strings.TrimSpace(trimmed[:idx])
			}
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				deps = append(deps, Dependency{
					Name:      parts[0],
					Version:   parts[1],
					Ecosystem: "go",
					Source:    f.Path,
				})
			}
		}
	}

	return deps
}

// parseRequirementsTxt parses Python pip dependencies from requirements.txt.
func parseRequirementsTxt(f *walker.FileNode) []Dependency {
	var deps []Dependency

	for _, line := range f.Lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Skip options like -r, --index-url, etc.
		if strings.HasPrefix(trimmed, "-") {
			continue
		}

		// Parse name==version, name>=version, name~=version, name
		var name, version string
		for _, op := range []string{"==", ">=", "<=", "!=", "~=", ">"} {
			if idx := strings.Index(trimmed, op); idx >= 0 {
				name = strings.TrimSpace(trimmed[:idx])
				version = strings.TrimSpace(trimmed[idx:])
				break
			}
		}
		if name == "" {
			name = trimmed
		}

		// Strip extras like package[extra]
		if idx := strings.Index(name, "["); idx >= 0 {
			name = name[:idx]
		}

		if name != "" {
			deps = append(deps, Dependency{
				Name:      strings.ToLower(name),
				Version:   version,
				Ecosystem: "pypi",
				Source:    f.Path,
			})
		}
	}

	return deps
}

// parseCargoToml parses Rust dependencies from Cargo.toml.
func parseCargoToml(f *walker.FileNode) []Dependency {
	var deps []Dependency
	inDeps := false
	inDevDeps := false

	for _, line := range f.Lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[dependencies]" {
			inDeps = true
			inDevDeps = false
			continue
		}
		if trimmed == "[dev-dependencies]" {
			inDevDeps = true
			inDeps = false
			continue
		}
		// Any new section ends the current block
		if strings.HasPrefix(trimmed, "[") && trimmed != "[dependencies]" && trimmed != "[dev-dependencies]" {
			inDeps = false
			inDevDeps = false
			continue
		}

		if (inDeps || inDevDeps) && strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), `"{}`)
				// Handle table-style: serde = { version = "1.0", features = [...] }
				if strings.Contains(version, "version") {
					if idx := strings.Index(version, `"`); idx >= 0 {
						end := strings.Index(version[idx+1:], `"`)
						if end >= 0 {
							version = version[idx+1 : idx+1+end]
						}
					}
				}
				deps = append(deps, Dependency{
					Name:      name,
					Version:   strings.Trim(version, `"`),
					Ecosystem: "cargo",
					Dev:       inDevDeps,
					Source:    f.Path,
				})
			}
		}
	}

	return deps
}

// parsePyprojectToml parses Python dependencies from pyproject.toml.
func parsePyprojectToml(f *walker.FileNode) []Dependency {
	var deps []Dependency
	inDeps := false

	for _, line := range f.Lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[tool.poetry.dependencies]" || trimmed == "[project]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") && inDeps {
			// Keep reading [project] until we hit a different section
			if trimmed != "[project]" {
				inDeps = false
			}
			continue
		}

		// dependencies = ["package>=1.0", ...]  (PEP 621 style)
		if strings.HasPrefix(trimmed, "dependencies") && strings.Contains(trimmed, "[") {
			// Extract items from the list
			start := strings.Index(trimmed, "[")
			end := strings.LastIndex(trimmed, "]")
			if start >= 0 && end > start {
				items := strings.Split(trimmed[start+1:end], ",")
				for _, item := range items {
					item = strings.Trim(strings.TrimSpace(item), `"'`)
					if item == "" {
						continue
					}
					name := item
					for _, op := range []string{">=", "==", "<=", "~="} {
						if idx := strings.Index(item, op); idx >= 0 {
							name = item[:idx]
							break
						}
					}
					deps = append(deps, Dependency{
						Name:      strings.ToLower(strings.TrimSpace(name)),
						Ecosystem: "pypi",
						Source:    f.Path,
					})
				}
			}
			continue
		}

		// Poetry style: package = "^1.0"
		if inDeps && strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), `"^~`)
				if name != "" && name != "python" {
					deps = append(deps, Dependency{
						Name:      name,
						Version:   version,
						Ecosystem: "pypi",
						Source:    f.Path,
					})
				}
			}
		}
	}

	return deps
}

// parsePomXML does a lightweight parse of Maven pom.xml dependencies.
// We avoid a full XML parser to keep the binary small — we scan for
// <dependency> blocks and extract groupId + artifactId + version.
func parsePomXML(f *walker.FileNode) []Dependency {
	var deps []Dependency
	var groupID, artifactID, version string
	inDep := false

	for _, line := range f.Lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "<dependency>") {
			inDep = true
			groupID, artifactID, version = "", "", ""
			continue
		}
		if strings.Contains(trimmed, "</dependency>") {
			if inDep && groupID != "" && artifactID != "" {
				deps = append(deps, Dependency{
					Name:      groupID + ":" + artifactID,
					Version:   version,
					Ecosystem: "maven",
					Source:    f.Path,
				})
			}
			inDep = false
			continue
		}

		if inDep {
			groupID = extractXMLTag(trimmed, "groupId", groupID)
			artifactID = extractXMLTag(trimmed, "artifactId", artifactID)
			version = extractXMLTag(trimmed, "version", version)
		}
	}

	return deps
}

// extractXMLTag extracts the text content of a simple XML tag from a line.
// Returns fallback if the tag is not found on this line.
func extractXMLTag(line, tag, fallback string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	start := strings.Index(line, open)
	end := strings.Index(line, close)
	if start >= 0 && end > start {
		return line[start+len(open) : end]
	}
	return fallback
}
