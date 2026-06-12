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
	// 1. Factory Code
	factoryPath := filepath.Join("pkg", "storage", "factory.go")
	factoryContent, err := os.ReadFile(factoryPath)
	if err == nil {
		content := string(factoryContent)
		caseStr := fmt.Sprintf("case \"%s\":\n\t\treturn New%sProvider(bucket), nil", data.LowerName, data.Name)
		if !strings.Contains(content, caseStr) {
			newCase := caseStr + "\n\tdefault:"
			content = strings.Replace(content, "default:", newCase, 1)
			os.WriteFile(factoryPath, []byte(content), 0644)
			fmt.Println("Registrado Provider na Factory em pkg/storage/factory.go")
		}
	}

	// 2. Factory Test
	testPath := filepath.Join("pkg", "storage", "factory_test.go")
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		writeTemplate(testPath, "factory_test_base.tpl", data)
	}

	testContent, err := os.ReadFile(testPath)
	if err == nil {
		content := string(testContent)
		testCase := fmt.Sprintf("t.Run(\"Success %s\", func(t *testing.T) {\n\t\tprovider, err := NewStorageProvider(\"%s\", \"bucket\")\n\t\tassert.NoError(t, err)\n\t\tassert.NotNil(t, provider)\n\t})", data.Name, data.LowerName)
		if !strings.Contains(content, "Success "+data.Name) {
			newTest := testCase + "\n\n\tt.Run(\"Unsupported Driver\""
			content = strings.Replace(content, "t.Run(\"Unsupported Driver\"", newTest, 1)
			os.WriteFile(testPath, []byte(content), 0644)
			fmt.Println("Registrado Teste na Factory em pkg/storage/factory_test.go")
		}
	}

	// 3. Media Module Registration in cmd/api/main.go
	mainPath := filepath.Join("cmd", "api", "main.go")
	mainContent, err := os.ReadFile(mainPath)
	if err == nil {
		content := string(mainContent)
		// Adicionar Import
		if !strings.Contains(content, "backend-go/internal/app/media") {
			newImport := "\"backend-go/internal/app/product\"\n\t\"backend-go/internal/app/media\""
			content = strings.Replace(content, "\"backend-go/internal/app/product\"", newImport, 1)
		}
		// Adicionar Rota
		if !strings.Contains(content, "media.RegisterRoutes") {
			newRoute := "product.RegisterRoutes(protected, database.DB)\n\t\tmedia.RegisterRoutes(protected)"
			content = strings.Replace(content, "product.RegisterRoutes(protected, database.DB)", newRoute, 1)
		}
		os.WriteFile(mainPath, []byte(content), 0644)
		fmt.Println("Registrado Módulo Media em cmd/api/main.go")
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Uso: go run tools/generator/storage/main.go <ProviderName>")
	}

	name := os.Args[1]
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	lowerName := strings.ToLower(name)

	// Diretorios
	storageDir := filepath.Join("pkg", "storage")
	mediaDir := filepath.Join("internal", "app", "media")
	os.MkdirAll(storageDir, 0755)
	os.MkdirAll(mediaDir, 0755)

	data := TemplateData{
		Name:      name,
		LowerName: lowerName,
	}

	// Arquivos do provider
	writeTemplate(filepath.Join(storageDir, lowerName+".go"), lowerName+".tpl", data)
	writeTemplate(filepath.Join(storageDir, lowerName+"_test.go"), "storage_test.tpl", data)

	// Arquivos do módulo Media (se não existirem)
	if _, err := os.Stat(filepath.Join(mediaDir, "service.go")); os.IsNotExist(err) {
		writeTemplate(filepath.Join(mediaDir, "service.go"), "media_service.tpl", data)
		writeTemplate(filepath.Join(mediaDir, "handler.go"), "media_handler.tpl", data)
		writeTemplate(filepath.Join(mediaDir, "routes.go"), "media_routes.tpl", data)
		writeTemplate(filepath.Join(mediaDir, "media_test.go"), "media_test.tpl", data)
		writeTemplate(filepath.Join(mediaDir, "main_test.go"), "main_test.tpl", data)
	}

	// Automação
	automateRegistration(data)

	fmt.Printf("\nDriver de storage '%s' e Módulo Media instalados com sucesso!\n", name)
}
