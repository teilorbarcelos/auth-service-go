.PHONY: dev test coverage coverage-html generate storage-driver infra-up infra-down infra-stop infra-clean db-up redis-up app-up app-down metrics-up metrics-stop metrics-down

# Variáveis
ENVIRONMENT ?= development
GO_TEST_FLAGS ?= -v

dev:
	@export PATH=$$PATH:$$(go env GOPATH)/bin && \
	if command -v air > /dev/null; then \
	    air; \
	elif [ -f $$(go env GOPATH)/bin/air ]; then \
	    $$(go env GOPATH)/bin/air; \
	else \
	    echo "Air não encontrado. Instalando..."; \
	    go install github.com/air-verse/air@latest; \
	    $$(go env GOPATH)/bin/air; \
	fi

test:
	@echo "Executando testes em ambiente de: test"
	@export ENVIRONMENT=test && go test $(GO_TEST_FLAGS) ./...

coverage:
	@echo "Gerando relatório de cobertura..."
	@export ENVIRONMENT=test && go test -count=1 -coverpkg=./... -coverprofile=coverage.out $$(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)
	@echo "\n--- Resumo de Cobertura ---"
	@go tool cover -func=coverage.out
	@echo "\n--- Linhas Não Cobertas ---"
	@go tool cover -func=coverage.out | grep -v "100.0%" || echo "Parabéns! 100% de cobertura atingida."

coverage-html:
	@export ENVIRONMENT=test && go test -count=1 -coverpkg=./... -coverprofile=coverage.out $$(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)
	@go tool cover -html=coverage.out

# Geradores
swagger:
	@echo "Gerando documentação Swagger..."
	@$(shell go env GOPATH)/bin/swag init -g cmd/api/main.go --parseDependency --parseInternal

generate:
	go run tools/generator/crud/main.go $(name)

storage-driver:
	go run tools/generator/storage/main.go $(name)

# Migrations
migrate-diff:
	@go run ariga.io/atlas/cmd/atlas@latest migrate diff $(name) --env gorm

migrate-up:
	@go run ariga.io/atlas/cmd/atlas@latest migrate apply --env gorm --url "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

# Infraestrutura
infra-up:
	docker compose -f docker-compose.infra.yml up -d

infra-stop:
	docker compose -f docker-compose.infra.yml stop

infra-down:
	docker compose -f docker-compose.infra.yml down

infra-clean:
	docker compose -f docker-compose.infra.yml down -v --rmi all

db-up:
	docker compose -f docker-compose.infra.yml up -d db

redis-up:
	docker compose -f docker-compose.infra.yml up -d redis

# Métricas (Prometheus & Grafana)
metrics-up:
	@echo "📈 Subindo stack de métricas (Prometheus & Grafana)..."
	docker compose -f docker-compose.metrics.yml up -d

metrics-stop:
	@echo "🛑 Parando stack de métricas..."
	docker compose -f docker-compose.metrics.yml stop

metrics-down:
	@echo "🗑️ Removendo stack de métricas..."
	docker compose -f docker-compose.metrics.yml down
