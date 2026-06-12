package audit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"backend-go/internal/core/models"
	"gorm.io/gorm"
)

func uniqueLog(action string) *models.AuditLog {
	return &models.AuditLog{
		Action:      action,
		TargetTable: fmt.Sprintf("buf_test_%d", time.Now().UnixNano()),
		RecordID:    action,
		OldValues:   "{}",
		NewValues:   "{}",
	}
}

func TestAuditBuffer(t *testing.T) {
	t.Run("Push and Shutdown flush remaining entries", func(t *testing.T) {
		buf := NewAuditBuffer(testDB, 100, 1*time.Minute)
		defer buf.Shutdown()

		entry := uniqueLog("FLUSH_TEST")
		buf.Push(entry)
		buf.Shutdown()

		var found int64
		testDB.Model(&models.AuditLog{}).Where("table_name = ?", entry.TargetTable).Count(&found)
		assert.Equal(t, int64(1), found)
	})

	t.Run("Multiple Shutdown is safe", func(t *testing.T) {
		buf := NewAuditBuffer(testDB, 10, 1*time.Minute)
		buf.Shutdown()
		buf.Shutdown()
		buf.Shutdown()
	})

	t.Run("Batch size triggers flush", func(t *testing.T) {
		buf := NewAuditBuffer(testDB, 2, 1*time.Minute)
		defer buf.Shutdown()

		ref := uniqueLog("BATCH")
		for i := 0; i < 2; i++ {
			buf.Push(&models.AuditLog{
				Action:      ref.Action,
				TargetTable: ref.TargetTable,
				RecordID:    "",
				OldValues:   "{}",
				NewValues:   "{}",
			})
		}

		time.Sleep(100 * time.Millisecond)

		var count int64
		testDB.Model(&models.AuditLog{}).Where("table_name = ?", ref.TargetTable).Count(&count)
		assert.Equal(t, int64(2), count)
	})

	t.Run("Flush interval triggers flush", func(t *testing.T) {
		buf := NewAuditBuffer(testDB, 100, 50*time.Millisecond)
		defer buf.Shutdown()

		ref := uniqueLog("TIMER")
		buf.Push(&models.AuditLog{
			Action:      ref.Action,
			TargetTable: ref.TargetTable,
			RecordID:    "",
			OldValues:   "{}",
			NewValues:   "{}",
		})

		time.Sleep(150 * time.Millisecond)

		var count int64
		testDB.Model(&models.AuditLog{}).Where("table_name = ?", ref.TargetTable).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestSetAuditBuffer(t *testing.T) {
	t.Run("SetAuditBuffer enables async mode in hooks", func(t *testing.T) {
		originalBuf := auditBuffer
		auditBuffer = nil

		buf := NewAuditBuffer(testDB, 10, 100*time.Millisecond)
		SetAuditBuffer(buf)

		ctx := context.WithValue(context.Background(), "userID", "buf-test-user")
		db := testDB.WithContext(ctx)

		model := &AuditTestModel{Name: "Buffer Hook Test"}
		err := db.Create(model).Error
		assert.NoError(t, err)

		buf.Shutdown()
		SetAuditBuffer(originalBuf)

		var log models.AuditLog
		err = testDB.Where("record_id = ? AND action = ?", model.ID, "CREATE").First(&log).Error
		if assert.NoError(t, err) {
			assert.Equal(t, "buf-test-user", *log.UserID)
		}
	})
}

func TestAuditBuffer_Shutdown(t *testing.T) {
	t.Run("Shutdown on empty buffer", func(t *testing.T) {
		buf := NewAuditBuffer(testDB, 10, 1*time.Minute)
		buf.Shutdown()
	})

	t.Run("Push after Shutdown drops entry", func(t *testing.T) {
		ref := uniqueLog("DROP_AFTER")
		buf := NewAuditBuffer(testDB, 10, 1*time.Minute)
		buf.Shutdown()

		buf.Push(&models.AuditLog{
			Action:      ref.Action,
			TargetTable: ref.TargetTable,
			RecordID:    "",
		})

		var count int64
		testDB.Model(&models.AuditLog{}).Where("table_name = ?", ref.TargetTable).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

func TestAuditBuffer_FullBuffer(t *testing.T) {
	t.Run("Full buffer drops entry", func(t *testing.T) {
		ref := uniqueLog("FULL_BUF")
		buf := NewAuditBuffer(testDB, 10, time.Hour)
		buf.Shutdown()

		for i := 0; i < 21; i++ {
			buf.Push(&models.AuditLog{
				Action:      ref.Action,
				TargetTable: ref.TargetTable,
				RecordID:    fmt.Sprintf("%d", i),
				OldValues:   "{}",
				NewValues:   "{}",
			})
		}

		buf.drainAndFlush(nil)

		var count int64
		testDB.Model(&models.AuditLog{}).Where("table_name = ?", ref.TargetTable).Count(&count)
		assert.Equal(t, int64(20), count)
	})
}

func TestAuditBuffer_FlushError(t *testing.T) {
	t.Run("flush error is logged but not propagated", func(t *testing.T) {
		buf := &AuditBuffer{
			entries:   make(chan *models.AuditLog, 10),
			done:      make(chan struct{}),
			db:        testDB.Session(&gorm.Session{NewDB: true}),
			batchSize: 10,
		}
		buf.db.AddError(errors.New("db error"))

		// Via drainAndFlush
		buf.drainAndFlush([]*models.AuditLog{
			{Action: "ERR_FLUSH", TargetTable: "flush_err_test", RecordID: "1"},
		})
	})
}
