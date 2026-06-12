package audit

import (
	"sync"
	"time"

	"backend-go/internal/core/models"
	"backend-go/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuditBuffer struct {
	entries   chan *models.AuditLog
	done      chan struct{}
	flushTick *time.Ticker
	db        *gorm.DB
	batchSize int
	closeOnce sync.Once
	doneWg    sync.WaitGroup
}

func NewAuditBuffer(db *gorm.DB, batchSize int, flushInterval time.Duration) *AuditBuffer {
	b := &AuditBuffer{
		entries:   make(chan *models.AuditLog, batchSize*2),
		done:      make(chan struct{}),
		flushTick: time.NewTicker(flushInterval),
		db:        db,
		batchSize: batchSize,
	}
	b.doneWg.Add(1)
	go b.run()
	return b
}

func (b *AuditBuffer) Push(entry *models.AuditLog) {
	select {
	case b.entries <- entry:
	default:
		logger.Warn("audit buffer full, dropping entry")
	}
}

func (b *AuditBuffer) Shutdown() {
	b.closeOnce.Do(func() {
		b.flushTick.Stop()
		close(b.done)
	})
	b.doneWg.Wait()
}

func (b *AuditBuffer) run() {
	defer b.doneWg.Done()
	var batch []*models.AuditLog
	for {
		select {
		case entry := <-b.entries:
			batch = append(batch, entry)
			if len(batch) >= b.batchSize {
				b.flush(batch)
				batch = batch[:0]
			}
		case <-b.flushTick.C:
			if len(batch) > 0 {
				b.flush(batch)
				batch = batch[:0]
			}
		case <-b.done:
			b.drainAndFlush(batch)
			return
		}
	}
}

func (b *AuditBuffer) drainAndFlush(batch []*models.AuditLog) {
	for {
		select {
		case entry := <-b.entries:
			batch = append(batch, entry)
			if len(batch) >= b.batchSize {
				b.flush(batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				b.flush(batch)
			}
			return
		}
	}
}

func (b *AuditBuffer) flush(batch []*models.AuditLog) {
	if err := b.db.CreateInBatches(batch, b.batchSize).Error; err != nil {
		logger.Error("failed to flush audit batch", zap.Int("count", len(batch)), zap.Error(err))
	}
}
