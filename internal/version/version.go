// Package version carries build metadata injected via -ldflags at build time.
package version

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
