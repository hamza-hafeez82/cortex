package recon

import (
	"fmt"
	"sort"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/internal/walker"
)

// Runner executes Stage 1 reconnaissance against a RepoMap.
type Runner struct{}

// NewRunner creates a new recon Runner.
func NewRunner() *Runner {
	return &Runner{}
}

// Run performs the full reconnaissance pass and returns a ReconResult.
func (r *Runner) Run(repo *walker.RepoMap) (*ReconResult, error) {
	if repo == nil {
		return nil, fmt.Errorf("repo map is nil")
	}

	result := &ReconResult{
		RootPath:   repo.Root,
		TotalFiles: repo.TotalFiles,
		TotalLines: repo.TotalLines,
	}

	// Language statistics
	result.Languages = buildLanguageStats(repo)

	// Tech stack detection
	result.TechStack = DetectTechStack(repo)
	for _, t := range result.TechStack {
		result.Frameworks = append(result.Frameworks, t.Name)
	}

	// Dependency parsing
	result.Dependencies = ParseDependencies(repo)

	// Endpoint detection
	result.Endpoints = DetectEndpoints(repo)

	// Infrastructure flags
	result.HasDocker,
		result.HasDockerCompose,
		result.HasKubernetes,
		result.HasTerraform,
		result.HasCI,
		result.CIProviders = DetectInfrastructure(repo)

	return result, nil
}

// buildLanguageStats computes per-language statistics from the RepoMap.
// Only source languages are included; Unknown and config languages are
// grouped but not counted toward the percentage total.
func buildLanguageStats(repo *walker.RepoMap) []LanguageStats {
	var totalSourceLines int
	langLines := make(map[string]int)
	langFiles := make(map[string]int)

	for _, f := range repo.Files {
		langFiles[f.Language]++
		langLines[f.Language] += f.LineCount
		if parser.IsSourceLanguage(f.Language) {
			totalSourceLines += f.LineCount
		}
	}

	var stats []LanguageStats
	for lang, files := range langFiles {
		lines := langLines[lang]
		pct := 0.0
		if totalSourceLines > 0 && parser.IsSourceLanguage(lang) {
			pct = float64(lines) / float64(totalSourceLines) * 100
		}
		stats = append(stats, LanguageStats{
			Name:      lang,
			FileCount: files,
			LineCount: lines,
			Percent:   pct,
		})
	}

	// Sort by line count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].LineCount > stats[j].LineCount
	})

	return stats
}

// Summary returns a human-readable one-line summary of the recon result.
func (res *ReconResult) Summary() string {
	return fmt.Sprintf(
		"%d files · %d lines · %d languages · %d dependencies · %d endpoints",
		res.TotalFiles,
		res.TotalLines,
		len(res.Languages),
		len(res.Dependencies),
		len(res.Endpoints),
	)
}
