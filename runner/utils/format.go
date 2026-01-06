package utils

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// FormatDuration formats a duration to a human-readable string.
// Examples: "1h 23m 45s", "5m 30s", "45s", "123ms"
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}

// FormatBytes formats a byte count to a human-readable string with appropriate units.
// Examples: "1.5 GB", "256 MB", "64 KB", "128 B"
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatGas formats a gas value to a human-readable string.
// Examples: "50 Ggas", "1.5 Mgas", "500 Kgas"
func FormatGas(gas uint64) string {
	const (
		Kgas = 1000
		Mgas = Kgas * 1000
		Ggas = Mgas * 1000
	)

	switch {
	case gas >= Ggas:
		return fmt.Sprintf("%.2f Ggas", float64(gas)/float64(Ggas))
	case gas >= Mgas:
		return fmt.Sprintf("%.2f Mgas", float64(gas)/float64(Mgas))
	case gas >= Kgas:
		return fmt.Sprintf("%.2f Kgas", float64(gas)/float64(Kgas))
	default:
		return fmt.Sprintf("%d gas", gas)
	}
}

// FormatPercentage formats a ratio as a percentage string.
// Example: FormatPercentage(0.856, 1) returns "85.6%"
func FormatPercentage(ratio float64, precision int) string {
	percentage := ratio * 100
	return fmt.Sprintf("%.*f%%", precision, percentage)
}

// ParseDuration parses a human-readable duration string.
// Supports formats like "1h", "30m", "45s", "1h30m", "100ms"
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, errors.New("empty duration string")
	}

	// First try Go's built-in parser
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle simple numeric values (assume seconds)
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(val * float64(time.Second)), nil
	}

	return 0, errors.Errorf("unable to parse duration: %s", s)
}

// ParseGasLimit parses a gas limit string with optional suffixes.
// Examples: "50e9", "50000000000", "50G", "50Ggas"
func ParseGasLimit(s string) (uint64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, errors.New("empty gas limit string")
	}

	// Remove "gas" suffix if present
	s = strings.TrimSuffix(s, "gas")
	s = strings.TrimSpace(s)

	// Handle scientific notation (e.g., "50e9")
	if strings.Contains(s, "e") {
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, errors.Wrap(err, "failed to parse scientific notation")
		}
		return uint64(val), nil
	}

	// Handle suffixes (K, M, G, T)
	multipliers := map[string]uint64{
		"k": 1000,
		"m": 1000000,
		"g": 1000000000,
		"t": 1000000000000,
	}

	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to parse gas limit with suffix %s", suffix)
			}
			return uint64(val * float64(multiplier)), nil
		}
	}

	// Try parsing as plain number
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse gas limit")
	}

	return val, nil
}

// TruncateString truncates a string to the specified length, adding an ellipsis if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// TruncateMiddle truncates a string in the middle, preserving the start and end.
// Useful for displaying long hashes or addresses.
func TruncateMiddle(s string, startChars, endChars int) string {
	if len(s) <= startChars+endChars+3 {
		return s
	}
	return s[:startChars] + "..." + s[len(s)-endChars:]
}

// IsValidHexAddress checks if a string is a valid Ethereum address (0x + 40 hex chars).
func IsValidHexAddress(address string) bool {
	if len(address) != 42 {
		return false
	}
	if !strings.HasPrefix(address, "0x") {
		return false
	}
	matched, _ := regexp.MatchString("^0x[0-9a-fA-F]{40}$", address)
	return matched
}

// IsValidHexHash checks if a string is a valid 32-byte hex hash (0x + 64 hex chars).
func IsValidHexHash(hash string) bool {
	if len(hash) != 66 {
		return false
	}
	if !strings.HasPrefix(hash, "0x") {
		return false
	}
	matched, _ := regexp.MatchString("^0x[0-9a-fA-F]{64}$", hash)
	return matched
}

// CalculateStats calculates basic statistics for a slice of float64 values.
// Returns mean, standard deviation, min, max, and median.
func CalculateStats(values []float64) (mean, stdDev, min, max, median float64, err error) {
	if len(values) == 0 {
		return 0, 0, 0, 0, 0, errors.New("empty values slice")
	}

	// Calculate mean
	sum := 0.0
	min = values[0]
	max = values[0]
	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	stdDev = math.Sqrt(sumSquaredDiff / float64(len(values)))

	// Calculate median (copy and sort to avoid modifying original)
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sortFloat64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		median = (sorted[n/2-1] + sorted[n/2]) / 2
	} else {
		median = sorted[n/2]
	}

	return mean, stdDev, min, max, median, nil
}

// sortFloat64s sorts a slice of float64 in ascending order (simple insertion sort for small slices)
func sortFloat64s(values []float64) {
	for i := 1; i < len(values); i++ {
		key := values[i]
		j := i - 1
		for j >= 0 && values[j] > key {
			values[j+1] = values[j]
			j--
		}
		values[j+1] = key
	}
}

// FormatTimestamp formats a time.Time to a standardized string format.
func FormatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseTimestamp parses a timestamp string in various common formats.
func ParseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.Errorf("unable to parse timestamp: %s", s)
}

// SafeDivide performs division with zero-safety, returning 0 if divisor is 0.
func SafeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

// SafeDivideUint64 performs integer division with zero-safety.
func SafeDivideUint64(numerator, denominator uint64) uint64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}
