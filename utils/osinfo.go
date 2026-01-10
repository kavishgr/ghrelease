package utils

import (
	"runtime"
)

func OsInfo() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
