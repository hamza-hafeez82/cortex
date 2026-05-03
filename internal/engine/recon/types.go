package recon

// ReconResult is the complete output of Stage 1.
// It is built once and passed to all subsequent stages.
type ReconResult struct {
	// Tech stack
	Languages  []LanguageStats // languages detected, sorted by file count
	TechStack  []Technology    // frameworks, runtimes, tools detected
	Frameworks []string        // framework names only (convenience slice)

	// Dependencies
	Dependencies []Dependency // all dependencies across all manifests

	// Endpoints
	Endpoints []Endpoint // detected HTTP endpoints

	// Infrastructure
	HasDocker        bool
	HasDockerCompose bool
	HasKubernetes    bool
	HasTerraform     bool
	HasCI            bool
	CIProviders      []string // "github-actions", "gitlab-ci", "circleci", etc.

	// Summary
	TotalFiles int
	TotalLines int
	RootPath   string
}

// LanguageStats holds per-language file and line counts.
type LanguageStats struct {
	Name      string
	FileCount int
	LineCount int
	Percent   float64 // percentage of total source lines
}

// Technology represents a detected framework, runtime, or tool.
type Technology struct {
	Name       string // e.g. "React", "FastAPI", "Gin"
	Category   string // "framework", "runtime", "database", "tool"
	Language   string // primary language this tech belongs to
	Confidence string // "high", "medium", "low"
	Source     string // which file triggered detection
}

// Dependency represents a single package dependency.
type Dependency struct {
	Name      string // package name
	Version   string // declared version (may include range specifier)
	Ecosystem string // "npm", "pypi", "cargo", "go", "maven", etc.
	Dev       bool   // true if it's a dev/test dependency
	Source    string // relative path to the manifest file
}

// Endpoint represents a detected HTTP route.
type Endpoint struct {
	Method  string // GET, POST, PUT, DELETE, PATCH, * (any)
	Path    string // route path, e.g. "/api/users/:id"
	File    string // relative file path
	Line    int    // line number
	Handler string // function/handler name if detectable
}
