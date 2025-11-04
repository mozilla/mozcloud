package cmd

import "runtime/debug"

// getVersion return the application version
func getVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok || buildInfo.Main.Version == "" {
		return "development"
	} else {
		return buildInfo.Main.Version
	}
}
