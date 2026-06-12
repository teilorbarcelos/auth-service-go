package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		err := Do(func() error {
			attempts++
			return nil
		}, Config{MaxAttempts: 3, Delay: time.Millisecond, Factor: 1.0}, "test")
		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("success after retries", func(t *testing.T) {
		attempts := 0
		err := Do(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("not ready")
			}
			return nil
		}, Config{MaxAttempts: 5, Delay: time.Millisecond, Factor: 1.0}, "test")
		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("failure after all attempts", func(t *testing.T) {
		attempts := 0
		err := Do(func() error {
			attempts++
			return errors.New("always fails")
		}, Config{MaxAttempts: 3, Delay: time.Millisecond, Factor: 1.0}, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "3 tentativas")
		assert.Equal(t, 3, attempts)
	})
}
