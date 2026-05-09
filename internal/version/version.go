package version

// These variables are set at build time by GoReleaser via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a formatted version string.
func String() string {
	if Version == "dev" {
		return "dev (" + Commit[:min(7, len(Commit))] + ")"
	}
	return Version
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
