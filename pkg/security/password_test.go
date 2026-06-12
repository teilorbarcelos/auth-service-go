package security

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

type errorReader struct{}

func (errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestPasswordHashing(t *testing.T) {
	password := "my_secret_password"

	t.Run("HashPassword success", func(t *testing.T) {
		hash, err := HashPassword(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
		assert.Contains(t, hash, "$argon2id$")
	})

	t.Run("CheckPasswordHash success", func(t *testing.T) {
		hash, _ := HashPassword(password)
		isValid := CheckPasswordHash(password, hash)
		assert.True(t, isValid)
	})

	t.Run("CheckPasswordHash failure", func(t *testing.T) {
		hash, _ := HashPassword(password)
		isValid := CheckPasswordHash("wrong_password", hash)
		assert.False(t, isValid)
	})

	t.Run("CheckPasswordHash empty hash", func(t *testing.T) {
		isValid := CheckPasswordHash(password, "")
		assert.False(t, isValid)
	})

	t.Run("HashPassword rand read error", func(t *testing.T) {
		oldReader := CryptoReader
		CryptoReader = errorReader{}
		defer func() { CryptoReader = oldReader }()

		_, err := HashPassword(password)
		assert.Error(t, err)
	})

	t.Run("CheckPasswordHash bcrypt backward compatible", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if assert.NoError(t, err) {
			isValid := CheckPasswordHash(password, string(hash))
			assert.True(t, isValid)

			isValid = CheckPasswordHash("wrong", string(hash))
			assert.False(t, isValid)
		}
	})

	t.Run("checkArgon2 invalid format", func(t *testing.T) {
		assert.False(t, CheckPasswordHash(password, "$invalid$hash"))
		assert.False(t, CheckPasswordHash(password, "$argon2id$v=A$m=1,t=1,p=1$salt$hash"))
		assert.False(t, CheckPasswordHash(password, "$argon2id$v=19$invalid$salt$hash"))
		assert.False(t, CheckPasswordHash(password, "$argon2id$v=19$m=1,t=1,p=1$!!!$hash"))
		assert.False(t, CheckPasswordHash(password, "$argon2id$v=19$m=1,t=1,p=1$c2FsdA==$!!!"))
		assert.False(t, CheckPasswordHash(password, "$argon2id$v=19$m=1,t=1,p=1$c2FsdA$abc12345"))
		assert.False(t, CheckPasswordHash(password, "$argon2id$v=19$m=1,t=1,p=1$c2FsdA$!!!"))

		// Five parts instead of six
		assert.False(t, CheckPasswordHash(password, "$argon2id$v=19$m=1,t=1,p=1"))
	})

	t.Run("CheckPasswordHash wrong prefix", func(t *testing.T) {
		isValid := CheckPasswordHash(password, "$2a$10$invalid")
		assert.False(t, isValid)
	})
}
