package utils

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/pkg/errors"
)

// GenerateRandomID generates a random hexadecimal ID of the specified byte length.
func GenerateRandomID(bytes int) (string, error) {
	if bytes <= 0 {
		return "", errors.New("byte length must be positive")
	}

	randomBytes := make([]byte, bytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", errors.Wrap(err, "failed to generate random bytes")
	}
	return hex.EncodeToString(randomBytes), nil
}
