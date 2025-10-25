// Package minikanren provides a thread-safe parallel implementation of miniKanren in Go.
//
// Version: 0.9.0
//
// This package offers a complete set of miniKanren operators with high-performance
// concurrent execution capabilities, designed for production use.
package minikanren

// Version represents the current version of the GoKando miniKanren implementation.
const Version = "0.9.0"

// VersionInfo provides detailed version information.
type VersionInfo struct {
	Version    string `json:"version"`
	GoVersion  string `json:"go_version"`
	GitCommit  string `json:"git_commit,omitempty"`
	BuildDate  string `json:"build_date,omitempty"`
}

// GetVersion returns the current version string.
func GetVersion() string {
	return Version
}

// GetVersionInfo returns detailed version information.
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GoVersion: "1.18+",
	}
}