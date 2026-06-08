package common

import "strings"

// ParseRethVersionOutput extracts a "<version>-<7-char-sha>" identifier
// from the multi-line output emitted by `<reth-derivative> --version`.
// Both base-reth-node and reth-optimism-cli share the upstream
// reth-node-core build script, so the parser is hoisted here so the
// per-client wrappers don't drift.
//
// Modern format (reth ≥ v1.11.x), one field per line:
//
//	reth 1.11.3
//	Version: 1.11.3
//	Commit SHA: 2ac58a25f561827e2b816a3c1ed972194f3b2915
//	Build Timestamp: ...
//	Build Features: ...
//	Build Profile: maxperf
//
// Legacy single-line format (everything on one "Version: ... Commit
// SHA: ..." line) is still supported for older binaries.
//
// Returns "unknown" if the output is unparseable. The "-<sha>" suffix
// is dropped when no Commit SHA is present so the result remains
// useful for legacy outputs that only carry a semver.
//
// Distinguishing two builds at the same semantic version is the
// reason we capture the SHA at all — without it, the version-grouping
// mode in the comparison report would collapse separate builds into
// one bucket.
func ParseRethVersionOutput(output string) string {
	var version, sha string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if version == "" {
			if v := extractFieldValue(line, "Version:"); v != "" {
				version = strings.Fields(v)[0]
			}
		}
		if sha == "" {
			if s := extractFieldValue(line, "Commit SHA:"); s != "" {
				short := strings.Fields(s)[0]
				if len(short) > 7 {
					short = short[:7]
				}
				sha = short
			}
		}
		if version != "" && sha != "" {
			break
		}
	}
	switch {
	case version != "" && sha != "":
		return version + "-" + sha
	case version != "":
		return version
	default:
		return "unknown"
	}
}

func extractFieldValue(line, key string) string {
	idx := strings.Index(line, key)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(line[idx+len(key):])
	if rest == "" {
		return ""
	}
	return rest
}
