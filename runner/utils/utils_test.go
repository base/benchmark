package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateRandomID(t *testing.T) {
	t.Run("generates ID of correct length", func(t *testing.T) {
		id, err := GenerateRandomID(8)
		require.NoError(t, err)
		require.Len(t, id, 16) // 8 bytes = 16 hex chars
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		id1, err := GenerateRandomID(8)
		require.NoError(t, err)
		id2, err := GenerateRandomID(8)
		require.NoError(t, err)
		require.NotEqual(t, id1, id2)
	})

	t.Run("rejects zero bytes", func(t *testing.T) {
		_, err := GenerateRandomID(0)
		require.Error(t, err)
	})

	t.Run("rejects negative bytes", func(t *testing.T) {
		_, err := GenerateRandomID(-1)
		require.Error(t, err)
	})
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"seconds only", 45 * time.Second, "45s"},
		{"minutes and seconds", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"hours minutes seconds", 1*time.Hour + 23*time.Minute + 45*time.Second, "1h 23m 45s"},
		{"hours only", 2 * time.Hour, "2h"},
		{"zero", 0, "0ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1024, "1.00 KB"},
		{"megabytes", 1024 * 1024, "1.00 MB"},
		{"gigabytes", 1024 * 1024 * 1024, "1.00 GB"},
		{"terabytes", 1024 * 1024 * 1024 * 1024, "1.00 TB"},
		{"mixed", 1536 * 1024, "1.50 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatGas(t *testing.T) {
	tests := []struct {
		name     string
		gas      uint64
		expected string
	}{
		{"small gas", 500, "500 gas"},
		{"kilogas", 50000, "50.00 Kgas"},
		{"megagas", 5000000, "5.00 Mgas"},
		{"gigagas", 50000000000, "50.00 Ggas"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatGas(tt.gas)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPercentage(t *testing.T) {
	tests := []struct {
		name      string
		ratio     float64
		precision int
		expected  string
	}{
		{"zero", 0.0, 1, "0.0%"},
		{"half", 0.5, 1, "50.0%"},
		{"full", 1.0, 1, "100.0%"},
		{"with precision", 0.856, 2, "85.60%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPercentage(tt.ratio, tt.precision)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"go format", "1h30m", 1*time.Hour + 30*time.Minute, false},
		{"seconds", "45s", 45 * time.Second, false},
		{"milliseconds", "500ms", 500 * time.Millisecond, false},
		{"numeric seconds", "60", 60 * time.Second, false},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseGasLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
		wantErr  bool
	}{
		{"plain number", "50000000000", 50000000000, false},
		{"scientific", "50e9", 50000000000, false},
		{"with G suffix", "50G", 50000000000, false},
		{"with Ggas suffix", "50Ggas", 50000000000, false},
		{"with M suffix", "100M", 100000000, false},
		{"with K suffix", "500K", 500000, false},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGasLimit(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"no truncation", "short", 10, "short"},
		{"truncation", "this is a long string", 10, "this is..."},
		{"exact length", "exact", 5, "exact"},
		{"very short max", "hello", 2, "he"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateMiddle(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		startChars int
		endChars   int
		expected   string
	}{
		{"hash truncation", "0x1234567890abcdef1234567890abcdef", 6, 4, "0x1234...cdef"},
		{"short string", "0x1234", 6, 4, "0x1234"},
		{"address", "0x9855054731540A48b28990B63DcF4f33d8AE46A1", 10, 8, "0x98550547...d8AE46A1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateMiddle(tt.input, tt.startChars, tt.endChars)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidHexAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected bool
	}{
		{"valid lowercase", "0x9855054731540a48b28990b63dcf4f33d8ae46a1", true},
		{"valid mixed case", "0x9855054731540A48b28990B63DcF4f33d8AE46A1", true},
		{"too short", "0x1234", false},
		{"too long", "0x9855054731540a48b28990b63dcf4f33d8ae46a1aa", false},
		{"missing 0x", "9855054731540a48b28990b63dcf4f33d8ae46a1", false},
		{"invalid chars", "0x9855054731540a48b28990b63dcf4f33d8ae46g1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidHexAddress(tt.address)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidHexHash(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		expected bool
	}{
		{"valid", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", true},
		{"too short", "0x1234", false},
		{"address length", "0x9855054731540a48b28990b63dcf4f33d8ae46a1", false},
		{"missing 0x", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidHexHash(tt.hash)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateStats(t *testing.T) {
	t.Run("basic stats", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5}
		mean, stdDev, min, max, median, err := CalculateStats(values)
		require.NoError(t, err)
		require.Equal(t, 3.0, mean)
		require.Equal(t, 1.0, min)
		require.Equal(t, 5.0, max)
		require.Equal(t, 3.0, median)
		require.InDelta(t, 1.414, stdDev, 0.01)
	})

	t.Run("single value", func(t *testing.T) {
		values := []float64{42}
		mean, stdDev, min, max, median, err := CalculateStats(values)
		require.NoError(t, err)
		require.Equal(t, 42.0, mean)
		require.Equal(t, 42.0, min)
		require.Equal(t, 42.0, max)
		require.Equal(t, 42.0, median)
		require.Equal(t, 0.0, stdDev)
	})

	t.Run("empty slice", func(t *testing.T) {
		_, _, _, _, _, err := CalculateStats([]float64{})
		require.Error(t, err)
	})

	t.Run("even count median", func(t *testing.T) {
		values := []float64{1, 2, 3, 4}
		_, _, _, _, median, err := CalculateStats(values)
		require.NoError(t, err)
		require.Equal(t, 2.5, median)
	})
}

func TestSafeDivide(t *testing.T) {
	tests := []struct {
		name        string
		numerator   float64
		denominator float64
		expected    float64
	}{
		{"normal division", 10, 2, 5},
		{"zero divisor", 10, 0, 0},
		{"zero numerator", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeDivide(tt.numerator, tt.denominator)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeDivideUint64(t *testing.T) {
	tests := []struct {
		name        string
		numerator   uint64
		denominator uint64
		expected    uint64
	}{
		{"normal division", 10, 2, 5},
		{"zero divisor", 10, 0, 0},
		{"zero numerator", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeDivideUint64(tt.numerator, tt.denominator)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	ts := time.Date(2025, 1, 6, 12, 30, 45, 0, time.UTC)
	result := FormatTimestamp(ts)
	require.Equal(t, "2025-01-06T12:30:45Z", result)
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{"RFC3339", "2025-01-06T12:30:45Z", time.Date(2025, 1, 6, 12, 30, 45, 0, time.UTC), false},
		{"date only", "2025-01-06", time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC), false},
		{"space separator", "2025-01-06 12:30:45", time.Date(2025, 1, 6, 12, 30, 45, 0, time.UTC), false},
		{"invalid", "not-a-date", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTimestamp(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, tt.expected.Equal(result))
			}
		})
	}
}
