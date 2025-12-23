package utils

import (
	"log"
)

func SetRegex(ost, arch string) string {
	// Pattern breakdown:
	// (?i) - case insensitive
	// (?=.*pattern) - positive lookahead (must contain)
	// (?!.*pattern) - negative lookahead (must NOT contain)

	// Files we want to EXCLUDE:
	// - Checksums: .sha256, .sha256sum, checksums.txt
	// - SBOMs: .sbom
	// - Package formats: .rpm, .deb
	// - Other OSes: windows, freebsd, netbsd, openbsd, android

	// for android:
	// android-tools-macos-arm64.tar.gz  Matches (android is just in the name)
	// tool-android-aarch64.tar.gz  Excluded (android OS target)

	const (
		excludeCommon   = `(?!.*(?:\.sha256sum|\.sha256|\.sbom|checksums|\.txt))`
		excludePackages = `(?!.*(?:\.rpm|\.deb))`
		excludeOtherOS  = `(?!.*(?:freebsd|netbsd|openbsd|android))`
	)

	switch {
	case ost == "darwin" && arch == "amd64":
		// Must contain: (apple|darwin|macos|mac) AND (amd64|x86_64|x64)
		// Must NOT contain: other OSes, checksums, packages
		return `(?i)` +
			`(?=.*(?:apple|darwin|macos|mac))` +
			`(?=.*(?:amd64|x86_64|x64))` +
			excludeCommon + excludeOtherOS +
			`(?!.*(?:linux|windows|win64))` +
			`.*`

	case ost == "linux" && arch == "amd64":
		// Must contain: linux AND (amd64|x86_64|x64)
		return `(?i)` +
			`(?=.*linux)` +
			`(?=.*(?:amd64|x86_64|x64))` +
			excludeCommon + excludePackages + excludeOtherOS +
			`(?!.*(?:windows|win64|apple|darwin|macos|mac))` +
			`.*`

	case ost == "darwin" && arch == "arm64":
		return `(?i)` +
			`(?=.*(?:apple|darwin|macos|mac))` +
			`(?=.*(?:arm64|aarch64))` +
			excludeCommon + excludeOtherOS +
			`(?!.*(?:linux|windows|win64))` +
			`.*`

	case ost == "linux" && arch == "arm64":
		return `(?i)` +
			`(?=.*linux)` +
			`(?=.*(?:arm64|aarch64))` +
			excludeCommon + excludePackages + excludeOtherOS +
			`(?!.*(?:windows|win64|apple|darwin|macos|mac))` +
			`.*`

	default:
		log.Fatalf(
			"OS or Architecture not supported: %s/%s\n"+
				"Supported: macOS and Linux (amd64/arm64)\n"+
				"Please file an issue at: https://github.com/kavishgr/ghrelease/issues",
			ost, arch,
		)
	}

	return ""
}

// func SetRegex(ost, arch string) string {
// 	var regex string
//
// 	// perl regex
// 	// works with github.com/dlclark/regexp2
// 	switch {
// 	// darwin amd64
// 	case (ost == "darwin" && arch == "amd64"):
// 		regex = `(?i)(?=.*(?:apple|darwin|macos|mac))(?=.*(?:amd64|x86_64|x64))(?!.*(?:freebsd|netbsd|openbsd|linux|windows|win64|.sha256sum|.sha256|.sbom|checksums|.txt))(?:.*(?:apple|darwin|macos|mac).*?(?:amd64|x86_64|x64)|(?:amd64|x86_64|x64).*?(?:apple|darwin|macos|mac))(?:[^a-z]|$)`
//
// 	// linux amd64
// 	case (ost == "linux" && arch == "amd64"):
// 		//good
// 		regex = `(?i)(?=.*(?:linux))(?=.*(?:amd64|x86_64|x64))(?!.*(?:freebsd|netbsd|openbsd|windows|win64|apple|darwin|macos|mac|.sha256sum|.sha256|.sbom|checksums|.txt|.rpm|.deb))(?:.*(?:linux).*?(?:amd64|x86_64|x64)|(?:amd64|x86_64|x64).*?(?:linux))(?:[^a-z]|$)`
//
// 	// darwin arm64
// 	case (ost == "darwin" && arch == "arm64"):
// 		regex = `(?i)(?=.*(?:apple|darwin|macos|mac))(?=.*(?:arm64|aarch64))(?!.*(?:freebsd|netbsd|openbsd|linux|windows|win64|.sha256sum|.sha256|.sbom|checksums|.txt))(?:.*(?:apple|darwin|macos|mac).*?(?:arm64|aarch64)|(?:arm64|aarch64).*?(?:apple|darwin|macos|mac))(?:[^a-z]|$)`
//
// 	// linux arm64
// 	case (ost == "linux" && arch == "arm64"):
// 		regex = `(?i)(?=.*(?:linux))(?=.*(?:arm64|aarch64))(?!.*(?:freebsd|netbsd|openbsd|windows|win64|apple|darwin|macos|mac|.sha256sum|.sha256|.sbom|checksums|.txt|.rpm|.deb))(?:.*(?:linux).*?(?:arm64|aarch64)|(?:arm64|aarch64).*?(?:linux))(?:[^a-z]|$)`
//
// 	default:
// 		msg1 := "OS or Architecture is not supported or not found in the regex pattern"
// 		msg2 := "File an issue or make a pull request for your OS and Arch"
// 		msg3 := "Will only list/download for macOS and Linux for the following architecture: "
// 		msg4 := "x86_64/amd64 and arm64"
// 		log.Fatalf("%v\n%v\n%v\n%v", msg1, msg2, msg3, msg4)
// 	}
// 	return regex
// }
