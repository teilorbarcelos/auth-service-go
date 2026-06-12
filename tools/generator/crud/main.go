package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

type TemplateData struct {
	Name      string
	LowerName string
}

func writeTemplate(outputPath, templateName string, data TemplateData) {
	tmplContent, err := templatesFS.ReadFile(filepath.Join("templates", templateName))
	if err != nil {
		log.Fatalf("Erro ao ler template %s: %v", templateName, err)
	}

	t, err := template.New(templateName).Parse(string(tmplContent))
	if err != nil {
		log.Fatalf("Erro ao parsear template %s: %v", templateName, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Fatalf("Erro ao executar template %s: %v", templateName, err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		log.Fatalf("Erro ao escrever arquivo %s: %v", outputPath, err)
	}
	fmt.Printf("Criado: %s\n", outputPath)
}

func automateRegistration(data TemplateData) {
	// 1. Registrar no AutoMigrate em pkg/database/db.go
	dbPath := filepath.Join("pkg", "database", "db.go")
	dbContent, err := os.ReadFile(dbPath)
	if err == nil {
		content := string(dbContent)
		if !strings.Contains(content, "&models."+data.Name+"{}") {
			newModel := fmt.Sprintf("&models.Product{},\n\t\t&models.%s{},", data.Name)
			content = strings.Replace(content, "&models.Product{},", newModel, 1)
			os.WriteFile(dbPath, []byte(content), 0644)
			fmt.Println("Registrado AutoMigrate em pkg/database/db.go")
		}
	}

	// 1b. Registrar no AutoMigrate em pkg/testutil/containers.go
	testUtilPath := filepath.Join("pkg", "testutil", "containers.go")
	testUtilContent, err := os.ReadFile(testUtilPath)
	if err == nil {
		content := string(testUtilContent)
		if !strings.Contains(content, "&models."+data.Name+"{}") {
			newModel := fmt.Sprintf("&models.Product{},\n\t\t&models.%s{},", data.Name)
			content = strings.Replace(content, "&models.Product{},", newModel, 1)
			os.WriteFile(testUtilPath, []byte(content), 0644)
			fmt.Println("Registrado AutoMigrate em pkg/testutil/containers.go")
		}
	}

	// 2. Registrar Rotas em cmd/api/main.go
	mainPath := filepath.Join("cmd", "api", "main.go")
	mainContent, err := os.ReadFile(mainPath)
	if err == nil {
		content := string(mainContent)
		// Adicionar Import
		if !strings.Contains(content, "backend-go/internal/app/"+data.LowerName) {
			newImport := fmt.Sprintf("\"backend-go/internal/app/product\"\n\t\"backend-go/internal/app/%s\"", data.LowerName)
			content = strings.Replace(content, "\"backend-go/internal/app/product\"", newImport, 1)
		}
		// Adicionar Rota
		if !strings.Contains(content, data.LowerName+".RegisterRoutes") {
			newRoute := fmt.Sprintf("product.RegisterRoutes(protected, database.DB)\n\t\t%s.RegisterRoutes(protected, database.DB)", data.LowerName)
			content = strings.Replace(content, "product.RegisterRoutes(protected, database.DB)", newRoute, 1)
		}
		os.WriteFile(mainPath, []byte(content), 0644)
		fmt.Println("Registrado Rotas em cmd/api/main.go")
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Uso: go run tools/generator/crud/main.go <ModuleName>")
	}

	name := os.Args[1]
	// Garantir que Name comece com letra maiúscula (PascalCase) para tipos Go
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	lowerName := strings.ToLower(name)

	// Diretório do módulo
	appDir := filepath.Join("internal", "app", lowerName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		log.Fatalf("Erro ao criar diretório %s: %v", appDir, err)
	}

	// Diretório do modelo
	modelDir := filepath.Join("internal", "core", "models")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		log.Fatalf("Erro ao criar diretório %s: %v", modelDir, err)
	}

	data := TemplateData{
		Name:      name,
		LowerName: lowerName,
	}

	// Arquivos do módulo
	writeTemplate(filepath.Join(appDir, "repository.go"), "repository.tpl", data)
	writeTemplate(filepath.Join(appDir, "service.go"), "service.tpl", data)
	writeTemplate(filepath.Join(appDir, "handler.go"), "handler.tpl", data)
	writeTemplate(filepath.Join(appDir, "routes.go"), "routes.tpl", data)
	writeTemplate(filepath.Join(appDir, "repository_test.go"), "repository_test.tpl", data)
	writeTemplate(filepath.Join(appDir, "service_test.go"), "service_test.tpl", data)
	writeTemplate(filepath.Join(appDir, "handler_test.go"), "handler_test.tpl", data)
	writeTemplate(filepath.Join(appDir, "main_test.go"), "main_test.tpl", data)

	// Arquivo do modelo
	writeTemplate(filepath.Join(modelDir, lowerName+".go"), "model.tpl", data)

	// Automação de registros
	automateRegistration(data)

	fmt.Printf("\nMódulo '%s' gerado e registrado com sucesso!\n", name)
}
