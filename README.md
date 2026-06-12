# 🚀 Go Modular Backend Boilerplate

[![Go Coverage](https://img.shields.io/badge/Coverage-100%25-brightgreen.svg)](#-testes-e-cobertura)
[![Go Report Card](https://goreportcard.com/badge/github.com/teilorbarcelos/backend-go)](https://goreportcard.com/report/github.com/teilorbarcelos/backend-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/teilorbarcelos/backend-go/actions/workflows/ci.yml/badge.svg)](https://github.com/teilorbarcelos/backend-go/actions/workflows/ci.yml)

Boilerplate Go seguindo **Clean Architecture** e **Modular Monolith**. Foco em simplicidade, segurança essencial e cobertura de testes — sem overengineering.

---

## 💎 O que vem incluído

*   **🛡️ Segurança essencial:** RBAC, JWT com invalidação via Redis, Rate Limiting, Auditoria automática, Security headers, body size limit, sanitização de erros em produção.
*   **🏗️ Geradores de Código:** Crie módulos CRUD ou drivers de Storage com CLI.
*   **🧪 100% de cobertura:** Garantida por **Testcontainers** (Postgres e Redis reais nos testes).
*   **📊 Observabilidade:** Prometheus, logs estruturados com `zap` e request ID propagado via context.
*   **📄 Geração de PDF:** Integração streaming com microserviço de PDF, com fallback para mock.
*   **🔒 Validações de produção:** JWT_SECRET, RateLimitMax/Window, TrustedProxies.

---

## 🛠️ Stack

*   **Linguagem:** Go 1.21+
*   **Web Framework:** Gin
*   **ORM:** GORM + pgx (PostgreSQL)
*   **Cache/Session:** Redis
*   **Mensageria:** RabbitMQ
*   **Documentação:** Swagger / OpenAPI
*   **Live Reload:** Air
*   **Geração de PDF:** React-PDF Service

---

## 🏗️ Arquitetura

**Modular Monolith** — cada domínio (User, Role, Product, Auth, etc.) vive em `internal/app/` e tem seus próprios handler/service/repository/rotas, pronto para ser extraído em microserviço se necessário.

```text
.
├── cmd/api/            # Ponto de entrada
├── internal/
│   ├── app/            # Módulos de domínio (auth, user, role, product, dashboard, media)
│   ├── core/           # Compartilhado (models, audit, repository, handler, middleware)
│   └── infra/          # Implementações de infraestrutura (session, pdf)
├── pkg/                # Reutilizáveis: config, database, security, cache, messaging, logger, retry
├── tools/              # Geradores (CRUD, storage driver)
└── docker-compose.yml
```

---

## 🚀 Setup Rápido

### 1. Pré-requisitos
- [Docker](https://www.docker.com/) e [Docker Compose](https://docs.docker.com/compose/)
- [Go](https://golang.org/dl/) 1.21+ (para os geradores)

### 2. Configuração
```bash
cp .env.example .env
```

### 3. Subir a Infraestrutura
Sobe Postgres, Redis e RabbitMQ:
```bash
make infra-up
```

### 4. Rodar a Aplicação
Live reload com Air (instala automaticamente na primeira vez):
```bash
make dev
```
API disponível em `http://localhost:8888`.

---

## 🔐 Configuração por Variáveis de Ambiente

Todas as variáveis ficam no `.env` (carregado via Viper). Em produção, defina o que difere do dev.

### Banco de Dados
| Variável | Default | Descrição |
|----------|---------|-----------|
| `DATABASE_URL` | `postgres://...` | DSN do PostgreSQL |
| `DB_MAX_OPEN_CONNS` | `50` | Conexões máximas abertas |
| `DB_MAX_IDLE_CONNS` | `10` | Conexões ociosas no pool |
| `DB_CONN_MAX_LIFETIME` | `30m` | Tempo máximo de vida de uma conexão |
| `DB_CONN_MAX_IDLE_TIME` | `5m` | Tempo máximo ocioso de uma conexão |
| `DB_STATEMENT_TIMEOUT` | `30000` | Timeout de query em ms (PostgreSQL `statement_timeout`) |
| `DB_IDLE_IN_TX_TIMEOUT` | `60000` | Timeout de transação ociosa em ms |

### Segurança
| Variável | Default | Validação em produção |
|----------|---------|------------------------|
| `JWT_SECRET` | — | **obrigatório >= 32 caracteres** |
| `JWT_ISSUER` | `backend-go` | — |
| `JWT_AUDIENCE` | `backend-go-api` | — |
| `JWT_ACCESS_EXPIRY` | `15m` | — |
| `JWT_REFRESH_EXPIRY` | `168h` (7d) | — |
| `RATE_LIMIT_MAX` | `100` | **obrigatório > 0** |
| `RATE_LIMIT_WINDOW` | `1m` | **obrigatório parseável** |
| `TRUSTED_PROXIES` | `""` (confia em todos) | Configure o range CIDR do seu LB |

### API
| Variável | Default | Descrição |
|----------|---------|-----------|
| `ENVIRONMENT` | `development` | `development` / `test` / `production` |
| `PORT` | `3000` | Porta HTTP |
| `HOST` | `0.0.0.0` | Host |
| `LOG_LEVEL` | `info` | — |
| `FIRST_USER` | `admin@email.com` | Seed do admin |
| `FIRST_PASSWORD` | `admin@123` | Seed do admin |
| `PDF_SERVICE_URL` | `http://localhost:8889` | URL do serviço de PDF |

---

## 🛡️ Segurança

- **JWT HS256** com claims `iss`/`aud`, access token de 15 min, refresh de 7 dias.
- **Invalidação de sessão O(1)** via versionamento no Redis (sem `SCAN`/`DEL`).
- **Rate Limiting atômico** via Redis Lua script, com headers `X-RateLimit-*`.
- **RBAC** baseado em bitset de permissões compilado no JWT — checagem O(1) por request.
- **Auditoria automática** via GORM hooks, escrita em batch assíncrono.
- **Argon2id** para hash de senhas (compatível com hashes bcrypt antigos).
- **Hash de senhas nunca logado** (campo `password` é scrubado).
- **Security headers** em todas as respostas: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 1; mode=block`.
- **Request body limit** de 10 MB (proteção contra DoS).
- **Errors sanitizados em produção** — detalhes internos nunca vazam.
- **NoRoute/NoMethod** retornam JSON tipado.
- **Swagger desabilitado em produção**.

---

## 📊 Observabilidade

- **Métricas Prometheus** em `/metrics` (`http_requests_total`, `http_request_duration_seconds`).
- **Logs estruturados** com `zap` em JSON, com `requestId` propagado via context (typed key `logger.RequestIDKey`).
- **Slow query log** no GORM com threshold de 100 ms em dev.
- **Health check** em `/health`.

---

## 🧪 Testes e Cobertura

A meta deste projeto é **100% de cobertura** em statements, branches, functions e lines.

```bash
make test          # roda todos os testes
make coverage      # gera resumo no terminal
make coverage-html # abre relatório HTML no browser
```

Os testes usam `testcontainers` para subir Postgres e Redis reais, garantindo que a integração com o banco é testada de verdade — não com mocks.

### Compliance E2E

Há uma suite externa de testes E2E em [mage-backend-compliance](../mage-backend-compliance) que valida o backend rodando como um todo (auth, RBAC, auditoria, rate limit, session invalidation, etc.).

---

## 🛠️ Geradores de Código

```bash
make generate name=product       # cria módulo CRUD completo (repo, service, handler, routes, testes)
make storage-driver name=cloudinary  # cria driver de storage novo
make swagger                     # regenera a documentação OpenAPI
```

---

## 📜 Comandos do Makefile

| Comando | Descrição |
|---------|-----------|
| `make dev` | Sobe o servidor com live reload (Air) |
| `make test` | Roda todos os testes |
| `make coverage` | Resumo de cobertura no terminal |
| `make coverage-html` | Relatório HTML |
| `make swagger` | Regenera OpenAPI/Swagger |
| `make generate name=X` | Gera módulo CRUD |
| `make storage-driver name=X` | Gera driver de storage |
| `make infra-up` | Sobe Postgres, Redis, RabbitMQ |
| `make infra-down` | Para os containers de infra |
| `make migrate-diff name=X` | Cria migration a partir de mudanças no model |
| `make migrate-up` | Aplica migrations |

---

## 📖 Documentação da API

Quando em `development`, a documentação interativa está disponível em:
- **Swagger UI:** `http://localhost:8888/v1/docs/index.html`
- **OpenAPI JSON:** `http://localhost:8888/api-docs/openapi.json`

Em produção, ambos os endpoints são desabilitados por padrão.

---

## 🔍 Quality Gate

Cada commit roda:
1. Suite completa de testes do projeto
2. Suite de compliance E2E (em repositório separado)
3. Scan SonarQube com quality gate (cobertura > 95%, zero `BUG`/`VULNERABILITY` críticos, duplicação < 3%)

---

## 📁 Estrutura de Domínio

Cada módulo em `internal/app/` segue o mesmo padrão:

```
internal/app/user/
├── handler.go         # HTTP handlers (Gin)
├── service.go         # Lógica de negócio
├── repository.go      # Acesso a dados (GORM)
├── routes.go          # Registro de rotas
├── handler_test.go    # Testes do handler
├── service_test.go    # Testes do service
└── repository_test.go # Testes do repository
```

Módulos incluídos: `auth`, `user`, `role`, `product`, `dashboard`, `media`.

---

Desenvolvido como ponto de partida limpo para projetos Go. Adicione complexidade conforme a necessidade real — não antes.
