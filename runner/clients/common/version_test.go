package common

import "testing"

func TestParseRethVersionOutput_ModernMultiLine(t *testing.T) {
	// Verbatim format from reth v1.11.4 (paradigmxyz/reth tag).
	// Confirmed via librarian against the upstream build.rs at that tag.
	out := `reth 1.11.3
Version: 1.11.3
Commit SHA: 2ac58a25f561827e2b816a3c1ed972194f3b2915
Build Timestamp: 2026-05-01T12:00:00.000000000Z
Build Features: jemalloc,asm-keccak
Build Profile: maxperf
`
	got := ParseRethVersionOutput(out)
	if got != "1.11.3-2ac58a2" {
		t.Fatalf("got %q, want %q", got, "1.11.3-2ac58a2")
	}
}

func TestParseRethVersionOutput_LegacySingleLine(t *testing.T) {
	out := "reth-optimism-cli Version: 1.7.0 Commit SHA: 9d56da53ec0ad60e229456a0c70b338501d923a5 Build Timestamp: 2025-09-15 Build Profile: maxperf\n"
	got := ParseRethVersionOutput(out)
	if got != "1.7.0-9d56da5" {
		t.Fatalf("got %q, want %q", got, "1.7.0-9d56da5")
	}
}

func TestParseRethVersionOutput_NoSha(t *testing.T) {
	out := "Version: 1.7.0\nBuild Profile: release\n"
	if got := ParseRethVersionOutput(out); got != "1.7.0" {
		t.Fatalf("got %q, want %q", got, "1.7.0")
	}
}

func TestParseRethVersionOutput_Garbage(t *testing.T) {
	out := "this is not a reth version string\nfoo bar\n"
	if got := ParseRethVersionOutput(out); got != "unknown" {
		t.Fatalf("got %q, want %q", got, "unknown")
	}
}

func TestParseRethVersionOutput_ShortShaUnpadded(t *testing.T) {
	// Defensive: SHA shorter than 7 chars must not panic the [:7] slice.
	out := "Version: 1.0.0\nCommit SHA: abc\n"
	if got := ParseRethVersionOutput(out); got != "1.0.0-abc" {
		t.Fatalf("got %q, want %q", got, "1.0.0-abc")
	}
}

func TestParseRethVersionOutput_Empty(t *testing.T) {
	if got := ParseRethVersionOutput(""); got != "unknown" {
		t.Fatalf("got %q, want %q", got, "unknown")
	}
}
