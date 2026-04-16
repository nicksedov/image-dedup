package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2 parameters tuned for interactive login (OWASP recommendations)
// Memory: 64MB, Iterations: 3, Parallelism: 2, Key length: 32, Salt length: 16
const (
	argon2Memory      = 64 * 1024 // 64 MB
	argon2Iterations  = 3
	argon2Parallelism = 2
	argon2KeyLength   = 32
	argon2SaltLength  = 16
)

// HashPassword hashes a password using argon2id and returns the encoded string
// Format: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLength)

	// Encode as: $argon2id$v=19$m=<memory>,t=<iterations>,p=<parallelism>$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory/1024, argon2Iterations, argon2Parallelism, b64Salt, b64Hash,
	)

	return encoded, nil
}

// VerifyPassword compares a password with its hash using constant-time comparison
func VerifyPassword(password, hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" {
		return false
	}

	// Parse parameters from the hash string
	// parts[2] = "v=19", parts[3] = "m=64,t=3,p=2"
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false
	}

	var memory, iterations, parallelism int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Hash the provided password with the same parameters
	hashBytes := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory*1024), uint8(parallelism), uint32(len(expectedHash)))

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(expectedHash, hashBytes) == 1
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
