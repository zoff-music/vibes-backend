package helper

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type adminPasswordParameters struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
}

type adminPasswordHash struct {
	Parameters adminPasswordParameters
	Salt       []byte
	Hash       []byte
}

func GenerateAdminPasswordHash(password string, pepper string) (string, error) {
	if pepper == "" {
		return "", fmt.Errorf("error generating admin password hash: pepper is required")
	}

	salt := make([]byte, adminPasswordSaltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("error generating admin password salt: %w", err)
	}

	parameters := adminPasswordParameters{
		Memory:      adminPasswordMemory,
		Iterations:  adminPasswordIterations,
		Parallelism: adminPasswordParallelism,
	}
	input := generateAdminPasswordInput(password, pepper)
	hash := argon2.IDKey(
		input,
		salt,
		parameters.Iterations,
		parameters.Memory,
		parameters.Parallelism,
		adminPasswordHashLength,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		parameters.Memory,
		parameters.Iterations,
		parameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

func VerifyAdminPassword(
	password string,
	pepper string,
	encodedHash string,
) (bool, error) {
	if pepper == "" {
		return false, fmt.Errorf("error verifying admin password: pepper is required")
	}

	parsed, err := parseAdminPasswordHash(encodedHash)
	if err != nil {
		return false, fmt.Errorf("error parsing admin password hash: %w", err)
	}

	input := generateAdminPasswordInput(password, pepper)
	actualHash := argon2.IDKey(
		input,
		parsed.Salt,
		parsed.Parameters.Iterations,
		parsed.Parameters.Memory,
		parsed.Parameters.Parallelism,
		adminPasswordHashLength,
	)
	matches := subtle.ConstantTimeCompare(actualHash, parsed.Hash) == 1

	return matches, nil
}

func generateAdminPasswordInput(password string, pepper string) []byte {
	digest := hmac.New(sha256.New, []byte(pepper))
	_, _ = digest.Write([]byte(password))

	hash := digest.Sum(nil)
	return hash
}

func parseAdminPasswordHash(encodedHash string) (*adminPasswordHash, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != adminPasswordPartCount || parts[1] != "argon2id" {
		return nil, fmt.Errorf("error invalid encoded admin password hash")
	}

	var version int
	parsedValues, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return nil, fmt.Errorf("error parsing admin password hash version: %w", err)
	}
	if parsedValues != 1 || version != argon2.Version {
		return nil, fmt.Errorf("error unsupported admin password hash version")
	}

	var parameters adminPasswordParameters
	parsedValues, err = fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&parameters.Memory,
		&parameters.Iterations,
		&parameters.Parallelism,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing admin password parameters: %w", err)
	}
	if parsedValues != adminPasswordParameterCount ||
		parameters.Memory != adminPasswordMemory ||
		parameters.Iterations != adminPasswordIterations ||
		parameters.Parallelism != adminPasswordParallelism {
		return nil, fmt.Errorf("error unsupported admin password parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("error decoding admin password salt: %w", err)
	}
	if len(salt) != adminPasswordSaltLength {
		return nil, fmt.Errorf("error invalid admin password salt length")
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, fmt.Errorf("error decoding admin password hash: %w", err)
	}
	if len(hash) != adminPasswordHashLength {
		return nil, fmt.Errorf("error invalid admin password hash length")
	}

	parsed := adminPasswordHash{
		Parameters: parameters,
		Salt:       salt,
		Hash:       hash,
	}

	return &parsed, nil
}

const adminPasswordMemory = 64 * 1024

const adminPasswordIterations = 3

const adminPasswordParallelism = 4

const adminPasswordSaltLength = 16

const adminPasswordHashLength = 32

const adminPasswordPartCount = 6

const adminPasswordParameterCount = 3
