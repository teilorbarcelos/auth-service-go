package database

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestRunSeed_Full(t *testing.T) {
	db := testDB
	
	// Limpar tabelas para garantir que o adminCount == 0 seja disparado
	db.Exec("DELETE FROM \"user\"")
	db.Exec("DELETE FROM role_feature")
	db.Exec("DELETE FROM feature")
	db.Exec("DELETE FROM role")

	// Rodar seed pela primeira vez (Criação)
	RunSeed(db)

	// Rodar seed pela segunda vez (Idempotência)
	RunSeed(db)
}

func TestRunSeed_Errors(t *testing.T) {
	db := testDB

	t.Run("Hash Error", func(t *testing.T) {
		origHash := hashPassword
		defer func() { hashPassword = origHash }()
		
		db.Exec("DELETE FROM \"user\"")
		hashPassword = func(password string) (string, error) {
			return "", errors.New("mock hashing error")
		}
		RunSeed(db)
	})

	t.Run("DB Errors", func(t *testing.T) {
		// Forçar erro na criação de Feature
		db.Callback().Create().Before("gorm:create").Register("test:err_feat", func(d *gorm.DB) {
			if d.Statement.Schema != nil && d.Statement.Schema.Table == "feature" {
				d.AddError(errors.New("db error feature"))
			}
		})
		defer db.Callback().Create().Remove("test:err_feat")

		// Forçar erro na criação de Role
		db.Callback().Query().Before("gorm:query").Register("test:err_role", func(d *gorm.DB) {
			if d.Statement.Schema != nil && d.Statement.Schema.Table == "role" {
				d.AddError(errors.New("db error role"))
			}
		})
		defer db.Callback().Query().Remove("test:err_role")

		// Forçar erro na criação de Admin
		db.Callback().Create().Before("gorm:create").Register("test:err_admin", func(d *gorm.DB) {
			if d.Statement.Schema != nil && d.Statement.Schema.Table == "user" {
				d.AddError(errors.New("db error admin"))
			}
		})
		defer db.Callback().Create().Remove("test:err_admin")

		db.Exec("DELETE FROM \"user\"")
		db.Exec("DELETE FROM role_feature")
		db.Exec("DELETE FROM feature")
		db.Exec("DELETE FROM role")
		
		RunSeed(db)
	})
}
