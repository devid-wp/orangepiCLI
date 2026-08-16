// Package buildinfo exposes release metadata embedded by the Go linker.
//
// Release builds set these variables with -ldflags -X. Defaults keep local
// development builds identifiable without requiring a build script.
package buildinfo

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
