package detectors

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/engine/recon"
	"github.com/hamza-hafeez82/cortex/internal/walker"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

type VulnerableDepsDetector struct{}

func (d *VulnerableDepsDetector) ID() string       { return "CX-DEP-001" }
func (d *VulnerableDepsDetector) Name() string     { return "Vulnerable Dependency" }
func (d *VulnerableDepsDetector) Category() string { return detector.CategoryDependency }
func (d *VulnerableDepsDetector) Severity() string { return detector.SeverityCritical }

// knownVulnerable maps "ecosystem:package" to version constraint + advisory info.
// This is a curated static list of high-impact CVEs.
// In production this would be backed by the OSV database API.
type vulnEntry struct {
	belowVersion string // affected if version contains string less than this
	severity     string
	cve          string
	description  string
}

var knownVulnerable = map[string]vulnEntry{
	// npm
	"npm:lodash":           {belowVersion: "4.17.21", severity: detector.SeverityCritical, cve: "CVE-2021-23337", description: "Prototype pollution via zipObjectDeep"},
	"npm:axios":            {belowVersion: "0.21.2", severity: detector.SeverityHigh, cve: "CVE-2021-3749", description: "ReDoS in axios SSRF protection"},
	"npm:express":          {belowVersion: "4.19.0", severity: detector.SeverityHigh, cve: "CVE-2024-29041", description: "Open redirect vulnerability"},
	"npm:jsonwebtoken":     {belowVersion: "9.0.0", severity: detector.SeverityCritical, cve: "CVE-2022-23529", description: "Remote code execution via malicious JWK"},
	"npm:minimist":         {belowVersion: "1.2.6", severity: detector.SeverityHigh, cve: "CVE-2021-44906", description: "Prototype pollution"},
	"npm:node-fetch":       {belowVersion: "2.6.7", severity: detector.SeverityHigh, cve: "CVE-2022-0235", description: "Exposure of sensitive information"},
	"npm:follow-redirects": {belowVersion: "1.14.8", severity: detector.SeverityMedium, cve: "CVE-2022-0536", description: "Sensitive information exposure"},
	"npm:qs":               {belowVersion: "6.10.3", severity: detector.SeverityHigh, cve: "CVE-2022-24999", description: "Prototype poisoning"},
	"npm:moment":           {belowVersion: "2.29.4", severity: detector.SeverityHigh, cve: "CVE-2022-31129", description: "ReDoS vulnerability"},
	"npm:tar":              {belowVersion: "6.1.9", severity: detector.SeverityHigh, cve: "CVE-2021-37701", description: "Arbitrary file creation/overwrite"},
	// Python
	"pypi:django":       {belowVersion: "3.2.19", severity: detector.SeverityHigh, cve: "CVE-2023-31047", description: "Potential bypass of file upload validation"},
	"pypi:flask":        {belowVersion: "2.2.5", severity: detector.SeverityMedium, cve: "CVE-2023-25577", description: "Potential DoS via multipart boundary"},
	"pypi:pillow":       {belowVersion: "9.3.0", severity: detector.SeverityHigh, cve: "CVE-2022-45199", description: "Denial of service via crafted file"},
	"pypi:requests":     {belowVersion: "2.31.0", severity: detector.SeverityMedium, cve: "CVE-2023-32681", description: "Proxy-Authorization header leak"},
	"pypi:cryptography": {belowVersion: "41.0.0", severity: detector.SeverityHigh, cve: "CVE-2023-38325", description: "NULL pointer dereference in PKCS12"},
	"pypi:pyyaml":       {belowVersion: "6.0", severity: detector.SeverityCritical, cve: "CVE-2022-1471", description: "Remote code execution via yaml.load"},
	// Go
	"go:golang.org/x/net":    {belowVersion: "v0.7.0", severity: detector.SeverityHigh, cve: "CVE-2023-24540", description: "Improper sanitization in HTML templates"},
	"go:golang.org/x/crypto": {belowVersion: "v0.1.0", severity: detector.SeverityHigh, cve: "CVE-2022-27191", description: "Crash in golang.org/x/crypto/ssh"},
	// Rust
	"cargo:openssl": {belowVersion: "0.10.48", severity: detector.SeverityHigh, cve: "CVE-2023-0286", description: "X.400 address type confusion"},
}

func (d *VulnerableDepsDetector) Run(ctx *detector.ScanContext) []detector.Issue {
	// Parse dependencies fresh from the repo
	deps := recon.ParseDependencies(ctx.Repo)
	return d.checkDeps(deps, ctx.Repo)
}

func (d *VulnerableDepsDetector) checkDeps(deps []recon.Dependency, repo *walker.RepoMap) []detector.Issue {
	var issues []detector.Issue

	for _, dep := range deps {
		key := dep.Ecosystem + ":" + strings.ToLower(dep.Name)
		vuln, ok := knownVulnerable[key]
		if !ok {
			continue
		}

		// Simple version check: if the declared version contains the vulnerable prefix
		// In production this would use semver comparison
		ver := strings.TrimLeft(dep.Version, "^~>=<!")
		if ver == "" || ver == "*" || isVersionBelow(ver, vuln.belowVersion) {
			issues = append(issues, detector.Issue{
				Code:       d.ID(),
				Title:      d.Name(),
				Message:    dep.Name + "@" + dep.Version + " is affected by " + vuln.cve + ": " + vuln.description + " — upgrade to " + vuln.belowVersion + " or later",
				File:       dep.Source,
				Line:       0,
				Severity:   vuln.severity,
				Confidence: detector.ConfidenceHigh,
				Category:   d.Category(),
				Snippet:    dep.Name + ": " + dep.Version,
			})
		}
	}

	return issues
}

// isVersionBelow does a simple string prefix comparison for semver.
// For production quality, replace with a proper semver library.
func isVersionBelow(declared, threshold string) bool {
	d := strings.TrimLeft(declared, "v")
	t := strings.TrimLeft(threshold, "v")
	// Compare major.minor.patch numerically where possible
	dp := strings.Split(d, ".")
	tp := strings.Split(t, ".")
	for i := 0; i < len(tp) && i < len(dp); i++ {
		if dp[i] < tp[i] {
			return true
		}
		if dp[i] > tp[i] {
			return false
		}
	}
	return false
}
