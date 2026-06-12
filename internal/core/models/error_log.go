package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ErrorLog struct {
	ID           string    `gorm:"type:varchar(40);primaryKey" json:"id"`
	IDUser       *string   `gorm:"type:varchar(40);column:id_user" json:"id_user"`
	Source       string    `gorm:"type:text" json:"source"`
	ErrorMessage string    `gorm:"type:text;column:error_message" json:"error_message"`
	ErrorData    string    `gorm:"type:text;column:error_data" json:"error_data"`
	CreatedAt    time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
}

func (ErrorLog) TableName() string {
	return "audit.tb_error_log"
}

func (e *ErrorLog) BeforeCreate(tx *gorm.DB) (err error) {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return
}
