// Package version provides build-time version information.
package version

import "fmt"

// These variables are set at build time via ldflags.
var (
	// Version is the semantic version (e.g., "1.0.0").
	Version = "dev"
	// Commit is the git commit SHA.
	Commit = "unknown"
	// Date is the build date in RFC3339 format.
	Date = "unknown"
)

// String returns a formatted version string.
func String() string {
	return fmt.Sprintf("cpm %s (commit: %s, built: %s)", Version, Commit, Date)
}
