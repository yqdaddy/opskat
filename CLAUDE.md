# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpsKat is an AI-first desktop application for managing remote infrastructure (SSH, databases, Redis). Built with **Wails v2** (Go 1.25 backend + React 19 frontend). The desktop app communicates via Wails IPC — there is no HTTP API.

Module: `github.com/opskat/opskat`

## Common Commands

### Development
```bash
make dev              # Wails dev mode with hot reload
make install          # Install frontend deps (pnpm)
```

### Build
```bash
make build            # Production build
make build-embed      # Production with embedded opsctl CLI
make build-cli        # Standalone opsctl CLI binary
make install-cli      # Install opsctl to GOPATH/bin
```

### Testing
```bash
make test                              # All Go tests
go test ./internal/ai/...             # Single package
go test ./internal/ai/ -run TestName  # Single test
cd frontend && pnpm test              # Frontend tests (vitest)
cd frontend && pnpm test:watch        # Frontend tests in watch mode
```

### Linting & Formatting
```bash
make lint             # golangci-lint (16 linters, 10m timeout)
make lint-fix         # golangci-lint with auto-fix
cd frontend && pnpm lint       # ESLint + Prettier
cd frontend && pnpm lint:fix   # ESLint auto-fix
```

### Plugin
```bash
make install-skill    # Register Claude Code opsctl plugin
```

## Architecture

### Backend (Go) — Layered Architecture

```
main.go (Wails entry)
  └─ internal/app/        App struct — Wails binding layer, all public methods exposed to frontend via IPC
       ├─ internal/service/    Business logic (16 service packages: ssh_svc, sftp_svc, ai_provider_svc, etc.)
       ├─ internal/repository/ Data access (12 repos with interface + impl pattern)
       └─ internal/model/      Domain entities
```

**Key subsystems:**
- `internal/ai/` — AI agent: provider abstraction (Anthropic/OpenAI), tool registry (41 tools), command policy checker, conversation runner, context compression, audit logging
- `internal/sshpool/` — SSH connection pool with Unix socket proxy for opsctl CLI
- `internal/connpool/` — Database/Redis tunnel management
- `internal/approval/` — Inter-process approval workflow (Unix socket between desktop app and opsctl)
- `internal/bootstrap/` — App initialization: database, credentials, migrations, auth tokens
- `internal/embedded/` — Embedded opsctl binary (build tag: `embed_opsctl`)
- `cmd/opsctl/` — Standalone CLI tool for remote operations, designed for AI assistant integration

**Repository pattern:** Each repo has an interface, a default singleton, and `Register()`/getter functions.

**Database:** GORM + SQLite, migrations in `/migrations/` using gormigrate.

**Credential encryption:** Argon2id KDF + AES-256-GCM, master key in OS keychain.

### Frontend (React + TypeScript)

Located in `frontend/`. Uses Vite 6 bundler, Tailwind CSS 4, shadcn/ui (Radix), Zustand 5 for state.

**No React Router** — uses a custom tab-based navigation system (`tabStore`). Tab types: terminal, ai, query, page, info.

**State stores** (`src/stores/`): One Zustand store per domain — assetStore, tabStore, terminalStore, aiStore, queryStore, sftpStore, shortcutStore, terminalThemeStore.

**Backend calls:** Generated Wails bindings in `frontend/wailsjs/`. Import from `wailsjs/go/app/App`. Real-time updates via `EventsOn()`.

**i18n:** i18next with `zh-CN` and `en` locales in `src/i18n/locales/`.

**Terminal:** xterm.js 6 with split-pane support.

**Tests:** Vitest + happy-dom + React Testing Library. Setup file mocks Wails runtime at `src/__tests__/setup.ts`.

### CI (`.github/workflows/ci.yml`)

Runs on PR and pushes to main/develop:
- Go: golangci-lint + `go test`
- Frontend: `pnpm lint` + `pnpm test` + `pnpm build`

### Git Commit Convention

Use **gitmoji** for commit messages. Common prefixes:
- ✨ New feature
- 🐛 Bug fix
- ♻️ Refactor
- 🎨 UI improvement
- ⚡️ Performance
- 🔒 Security
- 🔧 Configuration / tooling
- ✅ Tests
- 📄 Documentation
- 🚀 Deploy / release related

### Conventions

- Go mocks: generated with `go.uber.org/mock` in `mock_*/` subdirectories
- Go test assertions: goconvey + testify
- Frontend formatting: Prettier (120 char width, 2-space indent)
- Soft deletes via Status field (StatusActive=1, StatusDeleted=2), not GORM soft delete
- i18n keys namespaced under `"common"` — use `t("key.subkey")`
- Version info embedded via ldflags at build time
