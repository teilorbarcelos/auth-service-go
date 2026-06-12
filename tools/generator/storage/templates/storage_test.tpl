package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test{{.Name}}Provider(t *testing.T) {
	ctx := context.Background()
	provider := New{{.Name}}Provider("test-bucket")

	t.Run("Upload", func(t *testing.T) {
		url, err := provider.Upload(ctx, "test.txt", []byte("hello"))
		assert.NoError(t, err)
		assert.NotEmpty(t, url)
	})

	t.Run("GetURL", func(t *testing.T) {
		url, err := provider.GetURL(ctx, "test.txt")
		assert.NoError(t, err)
		assert.NotEmpty(t, url)
	})

	t.Run("Delete", func(t *testing.T) {
		err := provider.Delete(ctx, "test.txt")
		assert.NoError(t, err)
	})
}
