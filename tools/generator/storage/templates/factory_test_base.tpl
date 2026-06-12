package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStorageProvider(t *testing.T) {
	t.Run("Unsupported Driver", func(t *testing.T) {
		provider, err := NewStorageProvider("invalid", "bucket")
		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}
