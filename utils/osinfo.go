package utils

import (
	"runtime"
)

func OsInfo() (string, string) {
	os := runtime.GOOS
	arch := runtime.GOARCH
	return os, arch
}
