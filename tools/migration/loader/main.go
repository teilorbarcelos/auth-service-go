package main

import (
	"fmt"
	"io"
	"os"

	"backend-go/internal/core/models"

	"ariga.io/atlas-provider-gorm/gormschema"
)

func main() {
	stmts, err := gormschema.New("postgres").Load(
		&models.AuditLog{},
		&models.Role{},
		&models.Feature{},
		&models.RoleFeature{},
		&models.Auth{},
		&models.User{},
		&models.Product{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	io.WriteString(os.Stdout, stmts)
}
