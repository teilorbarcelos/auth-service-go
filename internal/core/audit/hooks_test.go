package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"backend-go/internal/core/models"
	"gorm.io/gorm"
)

type AuditTestModel struct {
	models.BaseModel
	Name string `gorm:"type:varchar(255)"`
}

func TestAuditHooks(t *testing.T) {
	ctx := context.WithValue(context.Background(), "userID", "test-user-id")
	db := testDB.WithContext(ctx)

	t.Run("auditCreateHook", func(t *testing.T) {
		// Test normal create
		model := &AuditTestModel{Name: "Create Test"}
		err := db.Create(model).Error
		assert.NoError(t, err)

		// Verify log
		var log models.AuditLog
		err = db.Where("table_name = ? AND record_id = ? AND action = ?", "audit_test_model", model.ID, "CREATE").First(&log).Error
		assert.NoError(t, err)
		assert.Contains(t, log.NewValues, "Create Test")

		// Test create with userID in context
		ctx := context.WithValue(context.Background(), "userID", "test-user-id")
		model2 := &AuditTestModel{Name: "Create Context Test"}
		err = db.WithContext(ctx).Create(model2).Error
		assert.NoError(t, err)

		var log2 models.AuditLog
		err = db.Where("record_id = ? AND action = ?", model2.ID, "CREATE").First(&log2).Error
		assert.NoError(t, err)
		assert.NotNil(t, log2.UserID)
		assert.Equal(t, "test-user-id", *log2.UserID)

		// Test ignoring audit_log table
		logRecord := &models.AuditLog{Action: "MANUAL", TargetTable: "audit.audit_log"}
		err = db.Create(logRecord).Error
		assert.NoError(t, err)
		
		var count int64
		db.Model(&models.AuditLog{}).Where("action = ? AND table_name = ?", "CREATE", "audit.audit_log").Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("auditCreateHook unauthenticated", func(t *testing.T) {
		dbUnauth := testDB // No context
		model := &AuditTestModel{Name: "Unauth Create Test"}
		err := dbUnauth.Create(model).Error
		assert.NoError(t, err)

		var count int64
		dbUnauth.Model(&models.AuditLog{}).Where("record_id = ?", model.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("auditUpdateHook", func(t *testing.T) {
		// 1. Update via struct
		modelStruct := &AuditTestModel{Name: "Before Update Struct"}
		db.Create(modelStruct)
		modelStruct.Name = "After Update Struct"
		err := db.Save(modelStruct).Error
		assert.NoError(t, err)

		var log models.AuditLog
		err = db.Where("record_id = ? AND action = ?", modelStruct.ID, "UPDATE").First(&log).Error
		assert.NoError(t, err)
		assert.Contains(t, log.OldValues, "Before Update Struct")
		assert.Contains(t, log.NewValues, "After Update Struct")

		// 2. Update via map
		modelMap := &AuditTestModel{Name: "Before Update Map"}
		db.Create(modelMap)
		err = db.Model(&AuditTestModel{}).Where("id = ?", modelMap.ID).Updates(map[string]interface{}{"ID": modelMap.ID, "name": "After Update Map"}).Error
		assert.NoError(t, err)

		var log2 models.AuditLog
		err = db.Where("record_id = ? AND action = ?", modelMap.ID, "UPDATE").First(&log2).Error
		assert.NoError(t, err)
		assert.Contains(t, log2.NewValues, "After Update Map")

		// 3. Update via map with ID included (to cover getRecordID map case)
		modelMapID := &AuditTestModel{Name: "Before Update Map ID"}
		db.Create(modelMapID)
		err = db.Model(&AuditTestModel{}).Where("id = ?", modelMapID.ID).Updates(map[string]interface{}{"ID": modelMapID.ID, "name": "After Update Map With ID"}).Error
		assert.NoError(t, err)

		var log3 models.AuditLog
		err = db.Where("record_id = ? AND action = ?", modelMapID.ID, "UPDATE").First(&log3).Error
		assert.NoError(t, err)
		assert.Contains(t, log3.NewValues, "After Update Map With ID")

		// 4. Update via map with lowercase 'id' (to cover DBName lookup)
		modelMapLower := &AuditTestModel{Name: "Before Update Map Lower"}
		db.Create(modelMapLower)
		err = db.Model(&AuditTestModel{}).Where("id = ?", modelMapLower.ID).Updates(map[string]interface{}{"id": modelMapLower.ID, "name": "Lower ID Test"}).Error
		assert.NoError(t, err)
	})

	t.Run("auditUpdateHook unauthenticated", func(t *testing.T) {
		dbUnauth := testDB // No context
		model := &AuditTestModel{Name: "Unauth Update Test"}
		db.Create(model) // db already has context with userID from TestAuditHooks definition

		model.Name = "Unauth Updated"
		err := dbUnauth.Save(model).Error
		assert.NoError(t, err)

		var count int64
		dbUnauth.Model(&models.AuditLog{}).Where("record_id = ? AND action = ?", model.ID, "UPDATE").Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("getRecordID multi-PK map", func(t *testing.T) {
		type MultiPKMapModel struct {
			ID1 string `gorm:"primaryKey"`
			ID2 string `gorm:"primaryKey"`
			Name string
		}
		db.AutoMigrate(&MultiPKMapModel{})
		
		model := &MultiPKMapModel{ID1: "M1", ID2: "M2", Name: "Multi"}
		db.Create(model)
		
		// Update with only one PK in map (to cover partial map cases in auditUpdateHook)
		err := db.Model(&MultiPKMapModel{}).Where("id1 = ? AND id2 = ?", "M1", "M2").Updates(map[string]interface{}{"ID1": "M1", "Name": "Updated"}).Error
		assert.NoError(t, err)
	})

	t.Run("getRecordID multi-PK", func(t *testing.T) {
		type MultiPKModel struct {
			ID1 string `gorm:"primaryKey"`
			ID2 string `gorm:"primaryKey"`
			Name string
		}
		db.AutoMigrate(&MultiPKModel{})
		
		model := &MultiPKModel{ID1: "1", ID2: "2", Name: "Multi"}
		db.Create(model)
		
		var log models.AuditLog
		err := db.Where("record_id = ? AND action = ?", "1:2", "CREATE").First(&log).Error
		assert.NoError(t, err)
	})

	t.Run("auditDeleteHook", func(t *testing.T) {
		model := &AuditTestModel{Name: "Delete Test"}
		db.Create(model)

		err := db.Delete(model).Error
		assert.NoError(t, err)

		var log models.AuditLog
		err = db.Where("record_id = ? AND action = ?", model.ID, "DELETE").First(&log).Error
		assert.NoError(t, err)
		assert.Equal(t, "audit_test_model", log.TargetTable)
	})

	t.Run("auditDeleteHook unauthenticated", func(t *testing.T) {
		dbUnauth := testDB // No context
		model := &AuditTestModel{Name: "Unauth Delete Test"}
		db.Create(model) // db already has context with userID from TestAuditHooks definition

		err := dbUnauth.Delete(model).Error
		assert.NoError(t, err)

		var count int64
		dbUnauth.Model(&models.AuditLog{}).Where("record_id = ? AND action = ?", model.ID, "DELETE").Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("getRecordID Error Paths", func(t *testing.T) {
		// Case: db.Statement.Schema is nil
		// This is hard to trigger with normal DB operations, but we can test the function directly
		dbEmpty := &gorm.DB{Statement: &gorm.Statement{}}
		id := getRecordID(dbEmpty)
		assert.Equal(t, "unknown", id)

		// Case: Empty ID
		model := &AuditTestModel{} // ID is empty before Create
		dbWithModel := db.Model(&AuditTestModel{}).Session(&gorm.Session{})
		dbWithModel.Statement.Dest = model
		id = getRecordID(dbWithModel)
		assert.Equal(t, "unknown", id)
	})
	
	t.Run("getUserIDFromContext Error Paths", func(t *testing.T) {
		// No context
		dbNoCtx := testDB.Session(&gorm.Session{})
		uid := getUserIDFromContext(dbNoCtx)
		assert.Nil(t, uid)

		// Context with wrong type
		ctx := context.WithValue(context.Background(), "userID", 123)
		dbWrongCtx := testDB.WithContext(ctx)
		uid = getUserIDFromContext(dbWrongCtx)
		assert.Nil(t, uid)
	})

	t.Run("getRecordID Map missing PK", func(t *testing.T) {
		dbSession := db.Session(&gorm.Session{})
		dbSession.Statement.Dest = map[string]interface{}{"Name": "No ID"}
		// Since we need a schema with primary fields, we'll use AuditTestModel's schema
		stmt := &gorm.Statement{DB: db, Dest: map[string]interface{}{"Name": "No ID"}}
		stmt.Parse(&AuditTestModel{})
		dbSession.Statement.Schema = stmt.Schema
		
		id := getRecordID(dbSession)
		assert.Equal(t, "unknown", id)
	})

	t.Run("auditUpdateHook with Map", func(t *testing.T) {
		model := &AuditTestModel{Name: "Map Update Test"}
		db.Create(model)
		
		// Update via map to trigger the map branch in auditUpdateHook
		err := db.Model(&AuditTestModel{}).Where("id = ?", model.ID).Updates(map[string]interface{}{"id": model.ID, "name": "Map Updated"}).Error
		assert.NoError(t, err)
		
		var log models.AuditLog
		err = db.Where("record_id = ? AND action = ?", model.ID, "UPDATE").Order("created_at desc").First(&log).Error
		assert.NoError(t, err)
		assert.Contains(t, log.NewValues, "Map Updated")
	})
}

func TestAuditHooks_ErrorPaths(t *testing.T) {
	ctx := context.WithValue(context.Background(), "userID", "test-user-id")
	db := testDB.WithContext(ctx)

	t.Run("Hooks with DB Error", func(t *testing.T) {
		dbWithErr := db.Session(&gorm.Session{})
		dbWithErr.Error = gorm.ErrInvalidData
		
		// Should return early
		auditCreateHook(dbWithErr)
		auditUpdateHook(dbWithErr)
		auditDeleteHook(dbWithErr)
	})

	t.Run("Hooks with nil Schema", func(t *testing.T) {
		dbNilSchema := db.Session(&gorm.Session{})
		dbNilSchema.Statement.Schema = nil
		
		// Should return early
		auditCreateHook(dbNilSchema)
		auditUpdateHook(dbNilSchema)
		auditDeleteHook(dbNilSchema)
	})

	t.Run("auditUpdateHook with unknown recordID", func(t *testing.T) {
		dbSession := testDB.Session(&gorm.Session{NewDB: true})
		dbSession.Statement.Dest = nil
		dbSession.Statement.Table = "some_table"
		
		// Manually set schema to pass the initial check
		stmt := &gorm.Statement{DB: testDB}
		stmt.Parse(&AuditTestModel{})
		dbSession.Statement.Schema = stmt.Schema
		
		// This should result in "unknown" recordID and return early at line 48
		auditUpdateHook(dbSession)
	})

	t.Run("Hooks ignore audit schema tables", func(t *testing.T) {
		// Test update to audit_log table
		logRecord := &models.AuditLog{Action: "UPDATE", TargetTable: "audit.audit_log", RecordID: "some-id"}
		db.Create(logRecord) // First create it
		
		logRecord.Action = "MANUAL"
		err := db.Save(logRecord).Error // This triggers auditUpdateHook
		assert.NoError(t, err)
		
		// Verify no log was created for this update
		var count int64
		db.Model(&models.AuditLog{}).Where("action = ? AND table_name = ? AND record_id = ?", "UPDATE", "audit.audit_log", "some-id").Count(&count)
		// It should only have 0 logs with action=UPDATE for the audit_log table
		assert.Equal(t, int64(0), count)

		// Test update to tb_error_log table
		errLogRecord := &models.ErrorLog{ID: "err-id", Source: "test", ErrorMessage: "err"}
		db.Create(errLogRecord)
		
		errLogRecord.ErrorMessage = "updated err"
		err = db.Save(errLogRecord).Error
		assert.NoError(t, err)

		var countErr int64
		db.Model(&models.AuditLog{}).Where("table_name = ? AND record_id = ?", "audit.tb_error_log", "err-id").Count(&countErr)
		assert.Equal(t, int64(0), countErr)
	})

	t.Run("auditUpdateHook old values not found", func(t *testing.T) {
		model := &AuditTestModel{BaseModel: models.BaseModel{ID: "non-existent"}}
		
		// We must include the ID in the updates map so that getRecordID can find it,
		// but since the record doesn't exist in the DB, query.Take will still fail.
		db.Model(model).Updates(map[string]interface{}{"ID": model.ID, "Name": "New"})
		
		var log models.AuditLog
		err := db.Where("record_id = ? AND action = ?", "non-existent", "UPDATE").First(&log).Error
		assert.NoError(t, err)
		assert.Equal(t, "{}", log.OldValues)
	})
}
