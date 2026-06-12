# Auth Service (Go) — TODO 🔐

Roadmap para extrair a autenticação do monolito `backend-go` em um microsserviço **plug-and-play** (porta `8001`), espelhando o que já foi feito no `auth-service-rust`. O objetivo é fragmentar o monólito sem dor: setar uma env no `backend-go` (`AUTH_MODE=remote`) passa a delegar `/v1/auth/*` para este serviço, que **compartilha o mesmo Postgres e Redis** do monólito.

> **Convenção desta lista**
> - Cada **fase** gera um commit próprio no `auth-service-go`. A **Fase 5** é o único commit previsto no `backend-go`.
> - A **bateria completa de testes** (unit, cobertura 100%, compliance e2e e Sonar) só roda na **Fase 6** — durante o caminho, cada fase valida com `go build ./...` e um smoke test manual.
> - Tudo aqui respeita **SOLID, DRY e clean code**: reuso integral de `pkg/*`, `internal/infra/session`, `internal/middleware/*` — zero duplicação com o monólito. A interface HTTP e a forma do JWT são idênticas às do `backend-go`, então o `Authenticate()` do monólito aceita tokens do service sem mudar uma linha.

---

## Fase 0 · Pré-requisitos (verificar antes de começar)

- [ ] **SonarQube** de pé em `localhost:9000` (hoje está **caído** — subir com `make up` dentro de `/home/teilor/MyProjects/sonar-qube` antes da Fase 6)
- [ ] **Postgres** do monólito de pé (`backend_go_db` em `localhost:5432`)
- [ ] **Redis** do monólito de pé (`backend_go_redis` em `localhost:6379`)
- [ ] `mage-backend-compliance` disponível em `/home/teilor/MyProjects/mage-boilerplates/mage-backend-compliance`
- [ ] `auth-service-rust` (8001) **parado** durante o desenvolvimento para não competir pela porta
- [ ] `backend-go` (8888) e `auth-service-go` (3000 hoje → `8001` depois) também parados enquanto o `auth-service-go` é montado
- [ ] `git checkout -b feat/auth-service-go` em `auth-service-go` (branch nova evita sujar a `main` durante o vai-e-vem)

---

## Fase 1 · Renomear o módulo Go

**Commit:** `chore: rename module backend-go → auth-service-go`

A pasta é cópia literal do `backend-go`. O módulo Go ainda diz `backend-go`. Antes de qualquer corte, o nome precisa refletir o projeto.

- [ ] `go.mod`: `module backend-go` → `module github.com/teilorbarcelos/auth-service-go`
- [ ] Trocar **todos** os `import "backend-go/..."` por `import "github.com/teilorbarcelos/auth-service-go/..."` nos `.go` (`rg '"backend-go' --type go` na raiz do projeto)
- [ ] `magerc.json` → ajustar `replacements[0].pattern` de `github.com/teilorbarcelos/backend-go` para `github.com/teilorbarcelos/auth-service-go`
- [ ] `sonar-project.properties` → `sonar.projectKey=teilorbarcelos_auth-service-go` e `sonar.projectName=Auth Service Go`
- [ ] Remover a linha `TODO.md` do `.gitignore` (este roadmap precisa ser commitável)
- [ ] Criar `.env` a partir de `.env.example`, ajustando:
  - `PORT=8001`
  - `JWT_SECRET=86941813-8b97-4cad-b0b2-f97734a947d7` (mesmo do `backend-go/.env`)
  - `DATABASE_URL=postgres://postgres:postgres@localhost:5432/backend_go?sslmode=disable`
  - `REDIS_URL=redis://localhost:6379/0`
  - `RABBITMQ_URL` pode ficar vazio/removido nesta fase (volta a ser útil só se algum módulo de infra exigir)
- [ ] Atualizar `.env.example` com os mesmos defaults
- [ ] `go mod tidy` + `go build ./...`

> **Validação:** `go build ./...` sem erro. Binário `cmd/api/main` é gerado.

---

## Fase 2 · Cortar tudo que não é auth

**Commit:** `feat: trim non-auth surface`

O monólito é grande. Cortar agora mantém o foco, reduz ruído no Sonar e torna 100% de cobertura viável. Tudo que sobrar tem que ser justificável para um microsserviço de auth.

### 2.1 — Remover

- [ ] `internal/app/dashboard`, `internal/app/media`, `internal/app/product`, `internal/app/role`, `internal/app/user` (não são auth; ficam no monólito)
- [ ] `pkg/email`, `pkg/messaging`, `pkg/storage`, `pkg/validator`
- [ ] `internal/infra/pdf` (PDF fica no monólito via `react-pdf-service`)
- [ ] `internal/core/audit` (auditoria é do monólito)
- [ ] `internal/core/handler/query_parser.go` (+ teste) — era para CRUD dinâmico
- [ ] `internal/middleware/metrics.go`, `internal/middleware/error_logger.go` (+ testes)
- [ ] `infra/metrics/` (Prometheus + Grafana) e `docker-compose.metrics.yml`
- [ ] `tools/generator/` (CRUD + storage) e `tools/migration/`
- [ ] `docs/` (swagger) e o binário compilado `api` na raiz
- [ ] Binário `coverage.out` e `tmp/` deixados de runs anteriores

### 2.2 — Manter (essenciais)

- [ ] `pkg/cache`, `pkg/config`, `pkg/database`, `pkg/logger`, `pkg/retry`, `pkg/security`, `pkg/testutil`
- [ ] `internal/core/models` (User, Auth, Role, Feature, RoleFeature, BaseModel, domainerr)
- [ ] `internal/core/repository` (base repository)
- [ ] `internal/infra/session` (vai ser usado para sincronizar `session:ver:%s`)
- [ ] `internal/middleware/auth.go`, `rbac.go`, `cors.go`, `ratelimit.go`, `logger.go`

### 2.3 — Limpar

- [ ] `pkg/database/db.go` → `AutoMigrate` apenas com `models.Role, Feature, RoleFeature, Auth, User` (remover `AuditLog, ErrorLog, Product`)
- [ ] `pkg/database/seeds.go` → manter só o essencial (4 features, role `administrator` com tudo, e o admin seed). Garantir idempotência (já usa `FirstOrCreate`)
- [ ] `cmd/api/main.go` → remover imports de `audit`, `messaging`, `dashboard`, `product`, `role`, `user`, `session` (este último some do main, fica só dentro do módulo auth). Registrar **apenas** o módulo `auth`. Trocar `r.Run(...)` direto, sem `auditBuffer`
- [ ] Remover `messaging.ConnectRabbitMQ()` do `main.go` (o service não usa fila)
- [ ] `go build ./...`

> **Validação:** `go build ./...` limpo. `make dev` sobe o binário em `8001` e responde `GET /health` 200, mesmo sem `/v1/auth/*` ainda funcional.

---

## Fase 3 · Adaptar o módulo `auth` para microsserviço

**Commit:** `feat: auth module → microservice`

O módulo `auth` já existe e cobre login/refresh/me/logout + password reset. O que muda é o **escopo**: ele vira a única fonte de `/v1/auth/*` e tem que falar o mesmo idioma do monólito (mesma assinatura de JWT, mesma convenção de `session:ver:%s` no Redis).

### 3.1 — Endpoints

- [ ] `internal/app/auth/routes.go`:
  - Manter públicos: `POST /v1/auth/login`, `POST /v1/auth/refresh`
  - Manter protegidos: `GET /v1/auth/me`, `POST /v1/auth/logout`
  - **Remover** (por ora) os endpoints de recuperação de senha: `/v1/auth/password/{request,validate,change}` — eles ficam no monólito
  - Adicionar `GET /v1/auth/.well-known/jwks.json` (placeholder RS256, mesmo do Rust)
  - `/health`, `/liveness`, `/ready` vão para um router separado `internal/observability` (mesmo padrão do Rust)

### 3.2 — Compatibilidade do JWT (ponto crítico)

O `backend-go/internal/middleware/auth.go` valida o token com:
- `HS256` + `JWT_SECRET` (compartilhado)
- Claims: `id`, `email`, `roleId`, `sessionVersion`, `permissions[]`, `iss`, `aud`, `exp`, `iat`
- Em seguida bate `session:ver:%s` no Redis com `claims.SessionVersion`

- [ ] `pkg/security/jwt.go` → **reaproveitar 100%**. O `GenerateToken` e `GenerateRefreshToken` do monólito já emitem esse formato. Nenhuma mudança aqui.
- [ ] `internal/app/auth/service.go` (`Login`):
  - Ler `SessionVersion` de `user.Auth.SessionVersion` (já existe no model)
  - Após gerar o access token, chamar `session.NewSessionManager().SetSessionVersion(ctx, user.ID, sessionVersion)` — espelha o que o monólito já faz
- [ ] `internal/app/auth/service.go` (`Logout`):
  - Chamar `session.NewSessionManager().InvalidateUserSessions(user.ID, "")` (bump do `session:ver:%s`) — exatamente igual ao monólito
- [ ] `internal/app/auth/service.go` (`Refresh`):
  - Validar JWT do refresh
  - Verificar `session:ver:%s` igual ao monólito
  - Gerar novo par, atualizar `session:ver:%s` (idempotente)
- [ ] Não reescrever `pkg/security/jwt.go` — reuso é SOLID-DRY

### 3.3 — Shape do payload de resposta

O compliance testa (`test_01_auth_session.py`) que `POST /v1/auth/login` retorna:

```json
{
  "token": "...",
  "refreshToken": "...",
  "user": {
    "id": "...",
    "email": "...",
    "role": { "id": "admin", "name": "Administrador", "permissions": [...] }
  }
}
```

- [ ] Ajustar `LoginResponse` em `internal/app/auth/service.go` para esse shape (campo `Valid` boolean pode coexistir; `User` precisa aninhar `role.permissions`)
- [ ] No service, popular `permissions` lendo de `user.Role.RoleFeature` (mesma origem do monólito — zero divergência)

### 3.4 — Inicialização / Seed

- [ ] No startup, se o banco estiver vazio, `RunSeed` cria as 4 features (`dashboard`, `user`, `role`, `product`), o role `administrator` com todas as permissões e o admin `admin@email.com / admin@123`
- [ ] Idempotente: `FirstOrCreate` (já é o padrão)
- [ ] `go build ./...`

> **Validação:** `make dev` → `curl -X POST localhost:8001/v1/auth/login -d '{"email":"admin@email.com","password":"admin@123"}'` retorna 200 com o shape acima. Decodificar o token em [jwt.io](https://jwt.io) deve mostrar `id`, `email`, `roleId`, `sessionVersion`, `permissions`, `iss`, `aud`.

---

## Fase 4 · Testes do microsserviço (100% cobertura)

**Commit:** `test: unit + integration coverage 100%`

Aqui os 100% viram. Estratégia reaproveitada do `auth-service-rust` (`tests/auth_integration.rs`): `testcontainers` para Postgres + Redis reais (mesmo padrão do `auth/main_test.go` atual), e cada teste cria usuário temporário com UUID único (zero race condition).

### 4.1 — Limpar

- [ ] `internal/app/auth/handler_test.go` → reescrever só com testes de validação de input (JSON inválido, campos faltando, 401 para credenciais erradas). Remover casos de password recovery
- [ ] `internal/app/auth/service_test.go` → remover casos de password recovery. Adicionar (mantendo o que já existe):
  - Login: sucesso, user não encontrado, user inativo, role inativo, auth inativo, senha errada
  - Refresh: sucesso, sessão expirada, user inativo
  - Logout invalida `session:ver:%s` (bump verificado no Redis)
  - Erro de hash/erro de token (substituindo `security.GenerateToken` via var injetada)

### 4.2 — Manter e cobrir

- [ ] `internal/infra/session/session_manager_test.go` (já está bom)
- [ ] `internal/middleware/auth_test.go` e `rbac_test.go`
- [ ] `pkg/security/jwt_test.go` e `pkg/security/password_test.go`

### 4.3 — Adicionar (HTTP integration)

- [ ] Subir `gin.Engine` real e bater via `httptest`:
  - `POST /v1/auth/login` (sucesso, 401, 400 sem campos)
  - `POST /v1/auth/refresh` (sucesso, 401 com refresh inválido)
  - `GET /v1/auth/me` (sem token → 401, com token → 200, com token revogado → 401)
  - `POST /v1/auth/logout` (revoga sessão)
  - `GET /health`, `/liveness`, `/ready`

> **Validação:** `go test -count=1 ./...` 100% verde. `make coverage` mostra 100% em todos os arquivos mantidos.

---

## Fase 5 · Adaptação do monolito `backend-go`

**Commit (no repo `backend-go`):** `feat: respect AUTH_MODE=remote to disable local auth routes`

Para o compliance rodar `make test-auth-go`, o monólito precisa, ao receber `AUTH_MODE=remote`, **simplesmente não registrar** `/v1/auth/*`. Nenhuma outra mudança. Middleware, RBAC, models, seed, session continuam idênticos.

- [ ] `backend-go/pkg/config/env.go` → adicionar `AuthMode string \`mapstructure:"AUTH_MODE"\`` (default `local`)
- [ ] `backend-go/cmd/api/main.go` → após ler config, se `AuthMode == "remote"`, pular o `auth.RegisterRoutes(v1, protected, database.DB)`. Manter tudo o resto idêntico
- [ ] `backend-go/.env.example` → documentar `AUTH_MODE=local` (default) e `AUTH_MODE=remote` (modo microsserviço)
- [ ] `backend-go/README.md` → seção "🔌 Modo microsserviço (auth-service)" explicando:
  - Como subir o `auth-service-go` (porta 8001)
  - Setar `AUTH_MODE=remote` no monólito
  - Diagrama do fluxo (mesmo do `auth-service-rust/README.md`)
  - Tabela "O que muda no monólito" (3 handlers removidos; middleware, RBAC e session inalterados)
- [ ] **Não tocar** em:
  - `internal/middleware/auth.go` (continua validando JWT — aceita tokens do service)
  - `internal/middleware/rbac.go`
  - `internal/infra/session/`
  - `internal/app/auth/*` (módulo continua existindo; só deixa de ser registrado quando `remote`)
  - `pkg/security/jwt.go` (é a mesma implementação que o service usa)
- [ ] `go build ./...` no monólito
- [ ] `go test ./...` no monólito continua passando (sem regressão)

> **Validação:** `AUTH_MODE=remote make dev` no monólito + `make dev` no `auth-service-go` → `curl localhost:8001/v1/auth/login` retorna 200, e o token funciona em `curl -H "Authorization: Bearer ..." localhost:8888/v1/auth/me`.

---

## Fase 6 · Bateria completa de validação (rodar **só no final**)

Conforme combinado, nada disso roda durante as fases 1–5 — só após a última commit.

### 6.1 — `auth-service-go`

- [ ] `make test:coverage` → 100% statements, branches, functions, lines
- [ ] `gofmt -l .` e `go vet ./...` limpos
- [ ] `make build` (binário compila)
- [ ] `make dev` → `/health` 200, `/ready` 200

### 6.2 — `backend-go`

- [ ] `make test:coverage` em modo `AUTH_MODE=local` → 100% como antes, sem regressão
- [ ] `AUTH_MODE=remote go test ./...` no monólito — confirma que os caminhos de skip não quebram nada

### 6.3 — Compliance E2E (ambos os modos)

- [ ] Em `mage-backend-compliance`:
  - `cp .env.go .env && make test-go` (modo **monolítico**) → 100% passando
  - `cp .env.auth.go .env && make test-auth-go` (modo **microsserviço**) → 100% passando
- [ ] Se algum teste falhar, **voltar e ajustar** — não pular teste

### 6.4 — SonarQube

- [ ] Confirmar `auth-service-go/sonar-project.properties`:
  - `sonar.projectKey=teilorbarcelos_auth-service-go`
  - `sonar.projectName=Auth Service Go`
  - `sonar.go.coverage.reportPaths=coverage.out`
- [ ] Confirmar `backend-go/sonar-project.properties` (sem mudança de key esperada)
- [ ] Rodar scanner nos dois e validar o quality gate:
  - Cobertura > 95% (alinhado ao gate atual do `backend-go`)
  - 0 bugs, 0 vulnerabilidades, 0 code smells novos
  - Duplicação < 3%

---

## Fase 7 · Finalização

- [ ] `auth-service-go/README.md` reescrito no padrão do `auth-service-rust/README.md`:
  - Arquitetura
  - Modo monólito vs. microsserviço
  - Tabela "O que muda no monólito"
  - Comandos do `make`
  - Seção de compliance
- [ ] Tag/release do `auth-service-go` (opcional)
- [ ] (Pós-MVP) considerar mover `/v1/auth/password/{request,validate,change}` do monólito para o service — não é pré-requisito do plug-and-play

---

## Resumo dos commits previstos

| # | Repo | Mensagem |
|---|------|----------|
| 1 | `auth-service-go` | `chore: rename module backend-go → auth-service-go` |
| 2 | `auth-service-go` | `feat: trim non-auth surface` |
| 3 | `auth-service-go` | `feat: auth module → microservice` |
| 4 | `auth-service-go` | `test: unit + integration coverage 100%` |
| 5 | `auth-service-go` | `docs: README + env example for plug-and-play` |
| 6 | `backend-go` | `feat: respect AUTH_MODE=remote to disable local auth routes` |
| 7 | `backend-go` | `docs: README seção modo microsserviço (auth-service-go)` |

> A bateria da Fase 6 (test/coverage/compliance/sonar) é apenas verificação — só vira commit se aparecer fix necessário durante ela.

---

## Princípios respeitados em todas as fases

- **S** Single Responsibility: cada módulo do service tem um papel (auth, jwt, session, rate limit).
- **O** Open/Closed: a interface `/v1/auth/*` é estável; mudanças internas não a afetam.
- **L** Liskov: tokens emitidos pelo service são 100% compatíveis com o validador do monólito.
- **I** ISP: o monólito só depende da interface HTTP do service; nada além.
- **D** DIP: o service depende de abstrações (`SessionStore`, `cache.RedisClient`, `Repository`).
- **DRY**: reuso integral de `pkg/security`, `pkg/cache`, `pkg/database`, `pkg/logger`, `internal/infra/session`, `internal/middleware/*` — zero duplicação com o monólito.
- **Clean Code**: nomes pequenos, sem comentários óbvios, funções coesas, side effects isolados no service.
