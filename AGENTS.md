# AGENTS.md

## Project

Self-hosted uptime monitoring app: Go backend (`cmd/uptime`, SQLite via `modernc.org/sqlite`) serving a REST API, and a React 19 + Vite + TypeScript frontend in `web/`. The Docker image is built from `Dockerfile-alpine` (nginx serves the SPA and proxies `/api` to the Go server on `:8080`).

## Pre-commit checks (MUST run before committing)

The lefthook pre-commit hook is broken in this WSL/Windows environment (it resolves a `C:\mnt\c\...` path and fails with `MODULE_NOT_FOUND` on `web/node_modules/lefthook/bin/index.js`). Do NOT rely on it. Run these checks manually, then commit with `--no-verify`:

From the repo root:

```bash
go build ./...
go vet ./...
go test ./...
```

From `web/`:

```bash
npx tsc --noEmit
npx prettier --check src/
npx eslint src/
npm test
```

These mirror the intended lefthook `pre-commit` commands (go-vet, tsc, eslint, prettier). Only commit once all pass.

## Other useful commands

```bash
# regenerate OpenAPI spec (after changing API/handlers/models)
go tool swag init -g cmd/uptime/main.go --output api

# build the frontend production bundle
npm run build        # in web/
```

## Gotchas

- The SPA source of truth is `.tsx`; stale `src/**/*.js` tsc-emit artifacts are gitignored but would shadow them in Vite resolution — delete any that appear.
- Auth: mutating endpoints require `Authorization: Bearer <token>`, token from `UPTIME_API_TOKEN` env (fails closed if unset). DB path is `UPTIME_DB_PATH` (default `./uptime.db`).
- Always commit regenerated `api/` files together with API changes.
