# Koyomi Gal

[**English**](README.md) | [**简体中文**](README.zh-CN.md)

A galgame community platform: catalog, ratings, resources, posts, articles, and admin tooling, built as two independent applications in one repository.

## Repository Layout

```
koyomi-gal/
├── apps/
│   ├── frontend/   # Nuxt 4 (Vue 3, TypeScript, Tailwind CSS 4, Pinia)
│   └── backend/    # Go 1.27 (Gin, GORM/PostgreSQL, Redis, Asynq, Swagger)
└── .github/
```

There is no root workspace manifest or task runner. Run all commands from the relevant app directory.

## Tech Stack

### Frontend (`apps/frontend`)

- Nuxt 4 / Vue 3 / TypeScript
- Tailwind CSS 4, `@kungal/ui` component library, Ant Design Vue
- Pinia for state management
- Orval-generated API client from the backend OpenAPI spec (`app/api/generated/`)

### Backend (`apps/backend`)

- Gin HTTP framework, GORM + pgx, PostgreSQL
- Redis + Asynq (async mail worker, separate process)
- JWT auth (access + refresh tokens), RBAC permission middleware
- Cloudflare R2 (S3-compatible) image storage
- Swagger docs served at `/swagger/index.html`
- SQL migrations embedded in the binary and applied on server startup (golang-migrate)

## Getting Started

### Prerequisites

- Go 1.27+
- Node.js 20+ and pnpm
- PostgreSQL and Redis
- An SMTP account (verification emails)
- Cloudflare R2 bucket (image assets)

### Backend

```bash
cd apps/backend
cp .env.development.example .env.development   # fill in real values
go run ./cmd/server   # HTTP API
go run ./cmd/worker   # async mail worker (required for verification emails)
```

- Both processes load `.env.<APP_ENV>` (default `APP_ENV=development`); existing environment variables override dotenv values.
- The server pings PostgreSQL and Redis before starting Gin and applies embedded migrations on startup.
- Server and worker must share the same Redis database and `VERIFICATION_SECRET` (base64, ≥ 32 decoded bytes). `CHANGE_ME` placeholders are rejected at startup.

### Frontend

```bash
cd apps/frontend
pnpm install --frozen-lockfile
cp .env.development.example .env.development   # set NUXT_PUBLIC_API_BASE to the backend origin
pnpm dev
```

- `NUXT_PUBLIC_API_BASE` is an origin only (e.g. `http://localhost:8080`); endpoint code appends `/api/v1`.
- Regenerate the API client after backend API changes: `pnpm api:generate` (requires the backend running with `/swagger/doc.json`).

## Commands

| App      | Command                          | Purpose                              |
| -------- | -------------------------------- | ------------------------------------ |
| Frontend | `pnpm dev`                       | Dev server                           |
| Frontend | `pnpm build`                     | Production build                     |
| Frontend | `pnpm exec nuxt typecheck`       | Type checking                        |
| Frontend | `pnpm api:generate`              | Regenerate API client via Orval      |
| Backend  | `go run ./cmd/server`            | Run HTTP server                      |
| Backend  | `go run ./cmd/worker`            | Run mail worker                      |
| Backend  | `go test ./...`                  | Run tests                            |
| Backend  | `go vet ./...`                   | Static analysis                      |
| Backend  | `bash scripts/gen-swagger.sh`    | Regenerate Swagger docs              |

## API Conventions

All API responses follow a `code` / `data` / `msg` envelope. API paths, payloads, auth cookies, CORS, and OpenAPI definitions are documented in the backend Swagger docs at `/swagger/index.html`.
