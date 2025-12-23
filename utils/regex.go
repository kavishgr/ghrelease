package utils

import (
	"log"
)

func SetRegex(ost, arch string) string {
	var regex string

	// perl regex
	// works with github.com/dlclark/regexp2
	switch {
	// darwin amd64
	case (ost == "darwin" && arch == "amd64"):
		regex = `(?i)(?=.*(?:apple|darwin|macos|mac))(?=.*(?:amd64|x86_64|x64))(?!.*(?:freebsd|netbsd|openbsd|linux|windows|win64|.sha256sum|.sha256|.sbom|checksums|.txt))(?:.*(?:apple|darwin|macos|mac).*?(?:amd64|x86_64|x64)|(?:amd64|x86_64|x64).*?(?:apple|darwin|macos|mac))(?:[^a-z]|$)`

	// linux amd64
	case (ost == "linux" && arch == "amd64"):
		//good
		regex = `(?i)(?=.*(?:linux))(?=.*(?:amd64|x86_64|x64))(?!.*(?:freebsd|netbsd|openbsd|windows|win64|apple|darwin|macos|mac|.sha256sum|.sha256|.sbom|checksums|.txt|.rpm|.deb))(?:.*(?:linux).*?(?:amd64|x86_64|x64)|(?:amd64|x86_64|x64).*?(?:linux))(?:[^a-z]|$)`

	// darwin arm64
	case (ost == "darwin" && arch == "arm64"):
		regex = `(?i)(?=.*(?:apple|darwin|macos|mac))(?=.*(?:arm64|aarch64))(?!.*(?:freebsd|netbsd|openbsd|linux|windows|win64|.sha256sum|.sha256|.sbom|checksums|.txt))(?:.*(?:apple|darwin|macos|mac).*?(?:arm64|aarch64)|(?:arm64|aarch64).*?(?:apple|darwin|macos|mac))(?:[^a-z]|$)`

	// linux arm64
	case (ost == "linux" && arch == "arm64"):
		regex = `(?i)(?=.*(?:linux))(?=.*(?:arm64|aarch64))(?!.*(?:freebsd|netbsd|openbsd|windows|win64|apple|darwin|macos|mac|.sha256sum|.sha256|.sbom|checksums|.txt|.rpm|.deb))(?:.*(?:linux).*?(?:arm64|aarch64)|(?:arm64|aarch64).*?(?:linux))(?:[^a-z]|$)`

	default:
		msg1 := "OS or Architecture is not supported or not found in the regex pattern"
		msg2 := "File an issue or make a pull request for your OS and Arch"
		msg3 := "Will only list/download for macOS and Linux for the following architecture: "
		msg4 := "x86_64/amd64 and arm64"
		log.Fatalf("%v\n%v\n%v\n%v", msg1, msg2, msg3, msg4)
	}
	return regex
}
