package audit

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"backend-go/internal/core/models"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const auditPrefix = "audit."

var auditBuffer *AuditBuffer

func SetAuditBuffer(buf *AuditBuffer) {
	auditBuffer = buf
}

func RegisterAuditHooks(db *gorm.DB) {
	db.Callback().Create().After("gorm:create").Register("audit:create", auditCreateHook)
	db.Callback().Update().Before("gorm:update").Register("audit:update", auditUpdateHook)
	db.Callback().Delete().Before("gorm:delete").Register("audit:delete", auditDeleteHook)
}

func writeAuditLog(db *gorm.DB, entry *models.AuditLog) {
	if auditBuffer != nil {
		auditBuffer.Push(entry)
		return
	}
	db.Session(&gorm.Session{NewDB: true}).Create(entry)
}

func auditCreateHook(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil || strings.HasPrefix(db.Statement.Schema.Table, auditPrefix) {
		return
	}

	recordID := getRecordID(db)
	newVals := models.MarshalValues(db.Statement.Dest)

	userID := getUserIDFromContext(db)
	if userID == nil {
		return
	}

	log := models.AuditLog{
		Action:      "CREATE",
		TargetTable: db.Statement.Schema.Table,
		RecordID:    recordID,
		OldValues:   "{}",
		NewValues:   newVals,
		UserID:      userID,
	}

	writeAuditLog(db, &log)
}

func auditUpdateHook(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil || strings.HasPrefix(db.Statement.Schema.Table, auditPrefix) {
		return
	}

	recordID := getRecordID(db)
	if recordID == "" || recordID == "unknown" {
		return
	}

	var oldValues map[string]interface{}
	query := db.Session(&gorm.Session{NewDB: true}).Table(db.Statement.Schema.Table)

	destValue := reflect.Indirect(reflect.ValueOf(db.Statement.Dest))
	for _, field := range db.Statement.Schema.PrimaryFields {
		val, _ := extractFieldValue(destValue, field, db.Statement.Context)
		if val != nil {
			query = query.Where(field.DBName+" = ?", val)
		}
	}

	if err := query.Take(&oldValues).Error; err != nil {
		oldValues = make(map[string]interface{})
	}

	newVals := models.MarshalValues(db.Statement.Dest)
	oldValsStr := models.MarshalValues(oldValues)

	userID := getUserIDFromContext(db)
	if userID == nil {
		return
	}

	log := models.AuditLog{
		Action:      "UPDATE",
		TargetTable: db.Statement.Schema.Table,
		RecordID:    recordID,
		OldValues:   oldValsStr,
		NewValues:   newVals,
		UserID:      userID,
	}

	writeAuditLog(db, &log)
}

func auditDeleteHook(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil || strings.HasPrefix(db.Statement.Schema.Table, auditPrefix) {
		return
	}

	recordID := getRecordID(db)

	userID := getUserIDFromContext(db)
	if userID == nil {
		return
	}

	log := models.AuditLog{
		Action:      "DELETE",
		TargetTable: db.Statement.Schema.Table,
		RecordID:    recordID,
		OldValues:   "{}",
		NewValues:   "{}",
		UserID:      userID,
	}

	writeAuditLog(db, &log)
}

func getRecordID(db *gorm.DB) string {
	if db.Statement.Schema == nil {
		return "unknown"
	}

	destValue := reflect.Indirect(reflect.ValueOf(db.Statement.Dest))
	var ids []string

	for _, field := range db.Statement.Schema.PrimaryFields {
		val, zero := extractFieldValue(destValue, field, db.Statement.Context)
		if !zero && val != nil {
			ids = append(ids, fmt.Sprintf("%v", val))
		}
	}

	if len(ids) > 0 {
		return strings.Join(ids, ":")
	}

	return "unknown"
}

func getUserIDFromContext(db *gorm.DB) *string {
	if ctxVal := db.Statement.Context.Value("userID"); ctxVal != nil {
		if id, ok := ctxVal.(string); ok && id != "" {
			return &id
		}
	}
	return nil
}

func extractFieldValue(destValue reflect.Value, field *schema.Field, ctx context.Context) (interface{}, bool) {
	if destValue.Kind() == reflect.Struct {
		return field.ValueOf(ctx, destValue)
	}
	if destValue.Kind() == reflect.Map {
		mapVal := destValue.MapIndex(reflect.ValueOf(field.Name))
		if !mapVal.IsValid() {
			mapVal = destValue.MapIndex(reflect.ValueOf(field.DBName))
		}
		if mapVal.IsValid() {
			return mapVal.Interface(), false
		}
		return nil, true
	}
	return nil, true
}
