# Auth Service (Go) 🔐

Microsserviço de autenticação dedicado, extraído do `backend-go`.
Plug-and-play com o monólito: setar `AUTH_MODE=remote` no monólito e subir este serviço na porta `8001`.

---

## 🚀 Tecnologias

- **Framework Web:** [Gin](https://github.com/gin-gonic/gin)
- **ORM:** [GORM](https://gorm.io/) + pgx (PostgreSQL)
- **Cache/Sessão:** Redis
- **Autenticação:** JWT (HS256) + Argon2id (compatível bcrypt)

---

## ✨ Funcionalidades

- **Login / Refresh / Logout** via JWT com refresh token rotation
- **RBAC no Redis:** Permissões cacheadas em `session:{user_id}:permissions`
- **Session Epoch:** Invalidação O(1) via `session:ver:%s` (INCR no Redis)
- **Rate Limiting:** Script Lua atômico no Redis
- **Health Checks:** `/health`, `/liveness`, `/ready`
- **JWKS Endpoint:** `GET /v1/auth/.well-known/jwks.json` (placeholder RS256)
- **Request Logging:** Log estruturado com zap

---

## 🔌 Plug-and-Play: Monolito → Microsserviço

O `auth-service-go` substitui o módulo de autenticação do `backend-go` sem alterar o middleware JWT, o RBAC ou a sessão Redis.

### Como funciona

```
FRONTEND                   AUTH SERVICE (8001)         MONOLITH (8888)
   │                            │                          │
   ├─ POST /login ────────────→│                          │
   │                            ├─ SELECT User+Auth+Role  │
   │                            ├─ Argon2id verify        │
   │                            ├─ Redis: create session  │
   │                            ├─ JWT (HS256)            │
   │←── { token, refresh } ────│                          │
   │                                                      │
   ├─ GET /orders (JWT) ────────────────────────────────→│
   │                                                      │
   │                          ├─ valida JWT local (HS256) │
   │                          ├─ checa session:ver:%s     │
   │                          ├─ RBAC check (bitset)      │
   │←─────────────────────────────────────────────────────│
```

### Modos de operação

#### Modo Monolítico (default)

O `backend-go` gerencia tudo — auth incluso. **Nenhuma configuração extra.**

```bash
AUTH_MODE=local    # (default) autenticação no próprio monólito
```

#### Modo Microsserviço (opt-in)

Auth extraído para o `auth-service-go`. O monólito mantém validação JWT + RBAC.

```bash
# backend-go/.env
AUTH_MODE=remote   # desliga /v1/auth/* no monólito

# auth-service-go/.env
JWT_SECRET=<mesma do monólito>
DATABASE_URL=<mesma do monólito>
REDIS_URL=<mesma do monólito>
```

### Passo a passo

```bash
# 1. Configure o auth-service
cd auth-service-go
cp .env.example .env
# Edite .env: mesma DATABASE_URL, JWT_SECRET e REDIS_URL do monólito

# 2. Suba o auth-service (porta 8001)
make dev

# 3. No monólito, ative o modo remoto
# backend-go/.env → AUTH_MODE=remote

# 4. Frontend passa a chamar:
#   - POST /v1/auth/login        → auth-service (8001)
#   - POST /v1/auth/refresh      → auth-service (8001)
#   - POST /v1/auth/logout       → auth-service (8001)
#   - Demais endpoints           → monólito (8888)

# 5. Pronto! O JWT emitido pelo auth-service é aceito pelo monólito.
```

### O que muda no monólito

| Componente | Antes (monolito) | Depois (auth-service) |
|---|---|---|
| `POST /v1/auth/login` | Handler local | ❌ Remove |
| `POST /v1/auth/refresh` | Handler local | ❌ Remove |
| `POST /v1/auth/logout` | Handler local | ❌ Remove |
| Middleware JWT | `ValidateToken(secret)` | ✅ **Igual** |
| Middleware RBAC | Lê bitset de permissões no JWT | ✅ **Igual** |
| Session version | `session:ver:%s` no Redis | ✅ **Igual** |

> **Apenas 3 handlers são removidos.** Todo o resto (middleware, RBAC, Redis) continua inalterado.

---

## 🏁 Começando

### Pré-requisitos

- Go 1.21+
- Docker (PostgreSQL + Redis)
- `backend-go` rodando (para criar as tabelas e dados iniciais)

### Setup

```bash
# 1. Suba infraestrutura
make infra-up
# ou use a mesma infra do monólito (recomendado)

# 2. Configure o ambiente
cp .env.example .env
# Edite DATABASE_URL, JWT_SECRET e REDIS_URL (mesmos do monólito)

# 3. Inicie o servidor
make dev
```

### Variáveis de ambiente

```bash
ENVIRONMENT=development
PORT=8001                   # Porta do auth-service
HOST=0.0.0.0

DATABASE_URL=postgres://postgres:postgres@localhost:5432/backend_go?sslmode=disable
REDIS_URL=redis://localhost:6379/0

JWT_SECRET=86941813-8b97-4cad-b0b2-f97734a947d7
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=1m
```

---

## 📡 Endpoints

| Método | Rota | Auth | Descrição |
|--------|------|------|-----------|
| POST | `/v1/auth/login` | ❌ | Login (email + password) |
| POST | `/v1/auth/refresh` | ❌ | Renova par de tokens |
| POST | `/v1/auth/logout` | ✅ | Revoga sessão |
| GET | `/v1/auth/me` | ✅ | Dados do usuário logado |
| GET | `/v1/auth/.well-known/jwks.json` | ❌ | JWKS (placeholder RS256) |
| GET | `/health` | ❌ | Health check |
| GET | `/liveness` | ❌ | Liveness probe |
| GET | `/ready` | ❌ | Readiness probe |

---

## 🧪 Testes

```bash
# Testes unitários e de integração
make test

# Cobertura
make coverage

# Linter
go vet ./...

# Formatação
gofmt -l .
```

### Compliance (E2E com monólito)

```bash
cd ../mage-backend-compliance

# Modo monolítico (auth no monólito)
cp .env.go .env
make test-go

# Modo microsserviço (auth no auth-service)
cp .env.auth.go .env
make test-auth-go
```

---

## 📊 Qualidade

- **Cobertura:** 100% em código de produção
- **SonarQube:** Quality Gate A (0 bugs, 0 vulnerabilidades, 0 code smells novos)
- **gofmt:** formatação padronizada
- **go vet:** zero warnings

---

## 🛠️ Comandos do Makefile

| Comando | Descrição |
|---------|-----------|
| `make dev` | Sobe o servidor com live reload (Air) |
| `make test` | Roda todos os testes |
| `make coverage` | Resumo de cobertura no terminal |
| `make build` | Compila binário de produção |
| `make infra-up` | Sobe Postgres e Redis |
| `make infra-down` | Para a infraestrutura |
