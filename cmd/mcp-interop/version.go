package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// These values are populated by release builds with -ldflags. Development and
// go-install builds fall back to Go module/VCS build information when present.
var (
	version   = ""
	commit    = ""
	buildDate = ""
)

type versionInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

func currentVersionInfo() versionInfo {
	info := versionInfo{
		Version:   strings.TrimSpace(version),
		Commit:    strings.TrimSpace(commit),
		BuildDate: strings.TrimSpace(buildDate),
	}

	if build, ok := debug.ReadBuildInfo(); ok {
		if info.Version == "" && build.Main.Version != "" && build.Main.Version != "(devel)" {
			info.Version = build.Main.Version
		}
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = setting.Value
				}
			case "vcs.time":
				if info.BuildDate == "" {
					info.BuildDate = setting.Value
				}
			}
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}
	if info.Commit == "" {
		info.Commit = "unknown"
	}
	if info.BuildDate == "" {
		info.BuildDate = "unknown"
	}
	return info
}

func formatVersion(info versionInfo) string {
	return fmt.Sprintf("mcp-interop %s (commit %s, built %s)", info.Version, shortCommit(info.Commit), info.BuildDate)
}

func shortCommit(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
