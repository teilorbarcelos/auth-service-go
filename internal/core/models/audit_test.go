package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalValues(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "{}",
		},
		{
			name:     "typed nil pointer (marshals to null)",
			input:    (*int)(nil),
			expected: "{}",
		},
		{
			name:     "invalid json input (channel)",
			input:    make(chan int),
			expected: "{}",
		},
		{
			name:     "valid map input",
			input:    map[string]string{"foo": "bar"},
			expected: `{"foo":"bar"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarshalValues(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditLog_BeforeCreate(t *testing.T) {
	t.Run("should generate ID if empty", func(t *testing.T) {
		audit := &AuditLog{}
		err := audit.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, audit.ID)
	})

	t.Run("should keep ID if not empty", func(t *testing.T) {
		id := "custom-id"
		audit := &AuditLog{ID: id}
		err := audit.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, id, audit.ID)
	})
}
