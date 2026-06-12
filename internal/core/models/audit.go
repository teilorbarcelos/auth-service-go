package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditLog struct {
	ID        string    `gorm:"type:varchar(40);primaryKey" json:"id"`
	Action    string    `gorm:"type:varchar(20);not null" json:"action"`      // CREATE, UPDATE, DELETE
	TargetTable string    `gorm:"type:varchar(100);not null;column:table_name" json:"table_name"`
	RecordID  string    `gorm:"type:varchar(40);not null" json:"record_id"`
	OldValues string    `gorm:"type:jsonb" json:"old_values"`
	NewValues string    `gorm:"type:jsonb" json:"new_values"`
	UserID    *string   `gorm:"type:varchar(40)" json:"user_id"`              // Quem fez a alteração
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit.audit_log"
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.OldValues == "" {
		a.OldValues = "{}"
	}
	if a.NewValues == "" {
		a.NewValues = "{}"
	}
	return
}

func MarshalValues(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return "{}"
	}
	return string(b)
}
