# Repository Guide

## Project boundaries

- There is no root workspace manifest or task runner. `apps/frontend` and `apps/backend` are independent projects; run commands from the relevant app directory.
- `apps/frontend` is a Nuxt 4 application managed by the local `pnpm-lock.yaml`. `apps/backend` is a standalone Go 1.27 module; there is no `go.work`.
- No CI workflows or pre-commit hooks exist, so run the relevant checks locally.

## Commands

Frontend (`apps/frontend`):

```bash
pnpm install --frozen-lockfile
pnpm dev
pnpm exec nuxt typecheck
pnpm build
```

- There are no frontend lint or test scripts/configs. `app/pages/test.vue` is a UI sandbox, not an automated test.
- `pnpm api:generate` reads `http://localhost:8080/swagger/doc.json`. The backend currently does not register that endpoint, so provide it before regenerating.

Backend (`apps/backend`):

```bash
go run ./cmd/server
go run ./cmd/worker
go test ./...
go vet ./...
```

- The server and mail worker are separate processes. `air` uses `.air.toml` and rebuilds only `./cmd/server`; Air itself is not pinned by this repo.
- Focus user-module verification with `go test ./internal/user/... ./pkg/...`; use `go test ./internal/user/service` for one package. There are currently no `*_test.go` files, so these commands provide compilation coverage until tests are added.

## Runtime and data

- Both Go entrypoints load `.env.<APP_ENV>` relative to the process working directory; unset `APP_ENV` means `.env.development`. Existing process variables override dotenv values.
- Treat `.env.*.example` files as the configuration inventory. Local `.env.*` files are ignored and may contain secrets; do not read or edit them unless explicitly requested.
- Backend secret placeholders named `CHANGE_ME` are invalid: secret loaders require standard base64 encoding of at least 32 decoded bytes.
- The HTTP server connects to and pings PostgreSQL and Redis before Gin starts, even for `/health`. The worker needs Redis and SMTP; verification emails are not delivered unless the worker is running.
- Server and worker must share the Redis database and decoded `VERIFICATION_SECRET` because verification jobs are encrypted before entering the `mail` Asynq queue.
- `internal/migrations/` contains raw SQL only. There is no migration runner or GORM `AutoMigrate`; starting the server never applies or reconciles schema changes.

## Architecture and contracts

- Backend HTTP dependency wiring belongs in `internal/app/app.go`; all HTTP routes belong in `internal/app/router.go`. Worker wiring starts in `cmd/worker/main.go`, with task handlers registered by `internal/infrastructures/queue/asynq.go`.
- Backend API handlers should use `pkg/response` and `pkg/errors` so responses follow the `code`/`data`/`msg` envelope and HTTP status mappings.
- Read `apps/frontend/docs/api-conventions.md` before changing API paths, payloads, auth cookies, CORS, or OpenAPI definitions. `NUXT_PUBLIC_API_BASE` is an origin only; endpoint code adds `/api/v1`.
- Never hand-edit `apps/frontend/app/api/generated/`. Orval writes there, while `app/api/mutator.ts` routes generated requests through the shared `$api` client.
- Frontend auth intentionally uses `useAuth` -> `services/auth.ts` -> injected `$api`, not generated auth methods. `$api` owns credential inclusion, bearer injection, refresh locking, and one-time `401` retry behavior.
- Nuxt pages are file-routed, but navigation is the static `navigationItems` list in `app/layouts/default.vue`; add both when a page should be discoverable.
- Reuse `AppPageContainer` for page framing and the auto-imported `Kun*` components. Dark mode depends on the `koyomi-color-mode` cookie and the `kun-dark-mode` class on `<html>`.
