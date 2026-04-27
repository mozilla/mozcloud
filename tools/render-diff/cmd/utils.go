package cmd

import "runtime/debug"

// Version is populated at build time via -ldflags "-X .../cmd.Version=...".
var Version string

// getVersion returns the application version. It prefers the ldflags-injected
// Version (used by goreleaser builds), then falls back to Go module build info
// (for `go install …@vX.Y.Z`), and finally returns "development".
func getVersion() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "development"
}
