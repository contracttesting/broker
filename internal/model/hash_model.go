package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Hash is the single, dynamic hash primitive for resource identity keys.
// Callers assemble the parts; the order of the parts is the identity.
func Hash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, ";;")))
	return hex.EncodeToString(sum[:])
}
