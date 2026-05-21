package consensus

import (
	"testing"
	"time"
)

func TestNextPayloadTimestampUsesNextBlockTime(t *testing.T) {
	now := time.Unix(100, int64(100*time.Millisecond))

	timestamp := nextPayloadTimestamp(100, now, 2*time.Second)

	if timestamp != 102 {
		t.Fatalf("expected next payload timestamp 102, got %d", timestamp)
	}
}

func TestNextPayloadTimestampSkipsTooCloseSlot(t *testing.T) {
	now := time.Unix(101, int64(250*time.Millisecond))

	timestamp := nextPayloadTimestamp(100, now, 2*time.Second)

	if timestamp != 104 {
		t.Fatalf("expected next payload timestamp 104, got %d", timestamp)
	}
}

func TestNextPayloadTimestampCatchesUpFromWallClock(t *testing.T) {
	now := time.Unix(120, 0)

	timestamp := nextPayloadTimestamp(100, now, 2*time.Second)

	if timestamp != 122 {
		t.Fatalf("expected next payload timestamp 122, got %d", timestamp)
	}
}
