package detector

import "github.com/hamza-hafeez82/cortex/internal/walker"

// Severity levels for issues.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// Category constants.
const (
	CategorySecurity     = "security"
	CategoryDependency   = "dependency"
	CategoryArchitecture = "architecture"
)

// Confidence levels for issues.
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// Issue represents a single finding produced by a detector.
type Issue struct {
	Code       string   // e.g. "CX-SEC-001"
	Title      string   // short human-readable title
	Message    string   // specific description for this finding
	File       string   // relative file path
	Line       int      // 1-indexed line number (0 if file-level)
	Severity   string   // critical, high, medium, low, info
	Confidence string   // high, medium, low
	Category   string   // security, dependency, architecture
	Snippet    string   // the offending line content
	Context    []string // surrounding lines for AI explanation
}

// ScanContext is passed to every detector during a scan.
// It contains everything a detector needs to do its job.
type ScanContext struct {
	Repo *walker.RepoMap
}

// Detector is the interface every security, dependency, and architecture
// detector must implement.
type Detector interface {
	// ID returns the unique issue code, e.g. "CX-SEC-001".
	ID() string

	// Name returns the human-readable detector name.
	Name() string

	// Category returns the detector category.
	Category() string

	// Severity returns the default severity for issues from this detector.
	Severity() string

	// Run executes the detector against the scan context and returns findings.
	Run(ctx *ScanContext) []Issue
}
