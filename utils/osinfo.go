package utils

import (
	"runtime"
)

// OsInfo returns the current operating system and architecture.
// Returns: (os, arch) where os is like "linux", "darwin" and
// arch is like "amd64", "arm64"
func OsInfo() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
