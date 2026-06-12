package database

import (
	"log"

	"backend-go/internal/core/models"
	"backend-go/pkg/config"
	"backend-go/pkg/security"

	"gorm.io/gorm"
)

var hashPassword = security.HashPassword

type FeatureData struct {
	Key         string
	Name        string
	Description string
}

type RoleFeatureData struct {
	FeatureKey string
	Create     bool
	View       bool
	Delete     bool
	Activate   bool
}

type RoleData struct {
	Key         string
	Name        string
	Description string
	Features    []RoleFeatureData
}

var Features = map[string]FeatureData{
	"dashboard": {Key: "dashboard", Name: "Dashboard", Description: "Visualizar indicadores e métricas do sistema"},
	"user":      {Key: "user", Name: "Usuários", Description: "Gerenciar usuários e acessos"},
	"role":      {Key: "role", Name: "Perfis de Acesso", Description: "Gerenciar cargos e permissões"},
	"product":   {Key: "product", Name: "Produtos", Description: "Gerenciar catálogo de produtos"},
}

var Roles = map[string]RoleData{
	"administrator": {
		Key:         "administrator",
		Name:        "Administrador",
		Description: "Acesso total ao sistema",
		Features: []RoleFeatureData{
			{FeatureKey: "dashboard", Create: true, View: true, Delete: true, Activate: true},
			{FeatureKey: "user", Create: true, View: true, Delete: true, Activate: true},
			{FeatureKey: "role", Create: true, View: true, Delete: true, Activate: true},
			{FeatureKey: "product", Create: true, View: true, Delete: true, Activate: true},
		},
	},
	"manager": {
		Key:         "manager",
		Name:        "Gerente",
		Description: "Gerente operacional",
		Features: []RoleFeatureData{
			{FeatureKey: "dashboard", Create: true, View: true, Delete: true, Activate: true},
			{FeatureKey: "user", Create: true, View: true, Delete: false, Activate: false},
			{FeatureKey: "role", Create: false, View: true, Delete: false, Activate: false},
			{FeatureKey: "product", Create: true, View: true, Delete: true, Activate: true},
		},
	},
	"operator": {
		Key:         "operator",
		Name:        "Operador",
		Description: "Operador de sistema",
		Features: []RoleFeatureData{
			{FeatureKey: "dashboard", Create: true, View: true, Delete: true, Activate: true},
			{FeatureKey: "user", Create: false, View: false, Delete: false, Activate: false},
			{FeatureKey: "role", Create: false, View: false, Delete: false, Activate: false},
			{FeatureKey: "product", Create: false, View: true, Delete: false, Activate: false},
		},
	},
}

func seedFeatures(db *gorm.DB) {
	for _, feat := range Features {
		var existing models.Feature
		if err := db.FirstOrCreate(&existing, models.Feature{
			BaseModel:   models.BaseModel{ID: feat.Key},
			Name:        feat.Name,
			Description: feat.Description,
		}).Error; err != nil {
			log.Printf("Erro ao fazer seed de Feature %s: %v", feat.Key, err)
		}
	}
}

func seedRoles(db *gorm.DB) {
	for _, r := range Roles {
		var role models.Role
		if err := db.Where(models.Role{BaseModel: models.BaseModel{ID: r.Key}}).FirstOrCreate(&role, models.Role{
			BaseModel:   models.BaseModel{ID: r.Key},
			Name:        r.Name,
			Description: r.Description,
		}).Error; err != nil {
			log.Printf("Erro ao fazer seed de Role %s: %v", r.Key, err)
			continue
		}

		for _, rf := range r.Features {
			var roleFeature models.RoleFeature
			db.Where(models.RoleFeature{IDRole: role.ID, IDFeature: rf.FeatureKey}).Assign(models.RoleFeature{
				Create:   rf.Create,
				View:     rf.View,
				Delete:   rf.Delete,
				Activate: rf.Activate,
			}).FirstOrCreate(&roleFeature)
		}
	}
}

func seedAdminUser(db *gorm.DB) {
	var adminCount int64
	db.Model(&models.User{}).Where("email = ?", config.AppConfig.FirstUserEmail).Count(&adminCount)

	if adminCount == 0 {
		log.Printf("Criando usuário administrador inicial: %s", config.AppConfig.FirstUserEmail)

		hashedPassword, err := hashPassword(config.AppConfig.FirstUserPassword)
		if err != nil {
			log.Printf("Erro ao hashear senha do administrador: %v", err)
		} else {
			adminUser := models.User{
				Name:   "Administrador",
				Email:  config.AppConfig.FirstUserEmail,
				Active: true,
				IDRole: "administrator",
				Auth: &models.Auth{
					Password:    &hashedPassword,
					Active:      true,
					FirstAccess: false,
				},
			}

			if err := db.Create(&adminUser).Error; err != nil {
				log.Printf("Erro ao criar usuário administrador: %v", err)
			} else {
				log.Println("Usuário administrador criado com sucesso!")
			}
		}
	}
}

func RunSeed(db *gorm.DB) {
	log.Println("Rodando script de inicialização do banco (Seed)...")

	seedFeatures(db)
	seedRoles(db)
	seedAdminUser(db)

	log.Println("Seed finalizado com sucesso!")
}
