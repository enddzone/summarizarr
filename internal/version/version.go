package version

import (
	"runtime"
	"runtime/debug"
)

var (
	// Version is the current version of the application
	Version = "dev"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
	// BuildTime is the time when the binary was built
	BuildTime = "unknown"
	// GoVersion is the version of Go used to build the binary
	GoVersion = runtime.Version()
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
	}
}

// GetVersion returns just the version string
func GetVersion() string {
	if Version == "dev" {
		// Try to get version from build info if available
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
					return "dev-" + setting.Value[:7]
				}
			}
		}
	}
	return Version
}
