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
