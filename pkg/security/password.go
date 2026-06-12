package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

var CryptoReader = rand.Reader

type Argon2Params struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

var DefaultArgon2Params = Argon2Params{
	Time:    3,
	Memory:  64 * 1024,
	Threads: 4,
	KeyLen:  32,
}

var errInvalidHash = errors.New("invalid argon2 hash format")

const argon2Prefix = "$argon2id$"

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(CryptoReader, salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, DefaultArgon2Params.Time, DefaultArgon2Params.Memory, DefaultArgon2Params.Threads, DefaultArgon2Params.KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, DefaultArgon2Params.Memory, DefaultArgon2Params.Time, DefaultArgon2Params.Threads, b64Salt, b64Hash), nil
}

func CheckPasswordHash(password, encodedHash string) bool {
	if strings.HasPrefix(encodedHash, argon2Prefix) {
		return checkArgon2(password, encodedHash)
	}
	return bcrypt.CompareHashAndPassword([]byte(encodedHash), []byte(password)) == nil
}

func checkArgon2(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false
	}

	var memory, timeVal uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeVal, &threads); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	expectedHash := argon2.IDKey([]byte(password), salt, timeVal, memory, threads, uint32(len(hash)))

	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}
