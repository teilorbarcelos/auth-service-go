package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSHA256(t *testing.T) {
	t.Run("Hash a simple string", func(t *testing.T) {
		text := "password"
		// echo -n "password" | sha256sum
		expected := "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"
		result := SHA256(text)
		assert.Equal(t, expected, result)
	})

	t.Run("Hash an empty string", func(t *testing.T) {
		text := ""
		// echo -n "" | sha256sum
		expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		result := SHA256(text)
		assert.Equal(t, expected, result)
	})
}
