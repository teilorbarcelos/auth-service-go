package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestErrorLog_TableName(t *testing.T) {
	e := ErrorLog{}
	assert.Equal(t, "audit.tb_error_log", e.TableName())
}

func TestErrorLog_BeforeCreate(t *testing.T) {
	t.Run("Generate ID if empty", func(t *testing.T) {
		e := ErrorLog{}
		err := e.BeforeCreate(&gorm.DB{})
		assert.NoError(t, err)
		assert.NotEmpty(t, e.ID)
		assert.Len(t, e.ID, 36) // UUID length
	})

	t.Run("Keep existing ID", func(t *testing.T) {
		existingID := "existing-id-123"
		e := ErrorLog{ID: existingID}
		err := e.BeforeCreate(&gorm.DB{})
		assert.NoError(t, err)
		assert.Equal(t, existingID, e.ID)
	})
}
