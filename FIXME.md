# FIXME: Recommended Improvements

This document tracks recommended improvements for the Uptime Monitor project. Items are grouped by area and prioritized.

---

## Backend / API

### High Priority
- [ ] **Incidents API** - `Incident` model exists in `internal/database/models.go` but no REST endpoints. Add `GET /api/targets/{id}/incidents` and `GET /api/incidents` for downtime history.
- [ ] **Pagination & Filtering** - `GET /api/targets` returns all targets (up to 1500 checks each). Add `?limit=&offset=` and `?status=up|down` query params.
- [ ] **Structured Logging** - Replace `log.Printf` with JSON logger (zerolog/zap) in `internal/service/polling.go` and handlers for observability.
- [ ] **Graceful Shutdown** - (Partially done) Handle SIGTERM in `main.go` to finish in-flight checks. Ensure polling service stops cleanly.

### Medium Priority
- [ ] **CORS Configuration** - Add CORS middleware if API consumed by external frontends.
- [ ] **Rate Limiting** - Add middleware for auth/abuse protection on mutating endpoints.
- [ ] **Config File Support** - Move from env-only to config file (YAML/TOML) for complex deployments.
- [ ] **Request Validation Middleware** - Centralize validation (currently duplicated in `internal/handler/validation.go` and handlers).
- [ ] **API Versioning** - Add `/api/v1/` prefix for future compatibility.
- [ ] **Context Timeouts** - Add `context.WithTimeout` to DB queries in `internal/database/targets.go`.

### Low Priority
- [ ] **Metrics Endpoint** - Add `/metrics` for Prometheus scraping (request latency, check counts, etc.).
- [ ] **Webhook Support** - Generic webhook notifications in addition to ntfy.

---

## Frontend

### High Priority
- [ ] **Delete Target UI** - Add delete button in edit modal (requires DELETE endpoint ✅).
- [ ] **Incident History View** - New page/modal showing downtime incidents (requires Incidents API).
- [ ] **Real-time Updates** - WebSocket/SSE for live status instead of 30s polling in `App.tsx`.

### Medium Priority
- [ ] **Dark Mode** - Add theme toggle (CSS variables already support it in `App.css`).
- [ ] **Keyboard Navigation** - Improve accessibility (focus management, ARIA labels).
- [ ] **Error Boundaries** - Add React error boundaries for graceful failure handling.

### Low Priority
- [ ] **Bulk Operations** - Multi-select for bulk delete/enable/disable.
- [ ] **Export/Import** - JSON/YAML export of targets for backup/migration.

---

## DevOps / Infrastructure

### High Priority
- [ ] **Docker Health Check** - Add `HEALTHCHECK` in `Dockerfile-alpine` using new `/healthz` endpoint ✅.
- [ ] **Multi-arch Docker** - Build for `linux/amd64,linux/arm64` in CI.

### Medium Priority
- [ ] **Dependency Scanning** - Add `govulncheck` and `npm audit` to CI pipeline.
- [ ] **SBOM Generation** - Generate Software Bill of Materials in CI.
- [ ] **Automated Releases** - Semantic release on merge to main.

### Low Priority
- [ ] **Helm Chart** - Package for Kubernetes deployment.
- [ ] **docker-compose.yml** - Add for local development with all services.

---

## Testing

### High Priority
- [ ] **Integration Tests** - Add end-to-end API tests (create → check → delete cycle).
- [ ] **Contract Tests** - Verify OpenAPI spec matches implementation.

### Medium Priority
- [ ] **E2E Tests** - Playwright/Cypress for critical user flows (login, add target, edit, delete).
- [ ] **Load Testing** - k6 scripts for polling service under load.

---

## Documentation

- [ ] **API Docs** - Host Swagger UI at `/swagger/` in production (currently only in dev).
- [ ] **Architecture Decision Records (ADRs)** - Document key decisions (SQLite, cron scheduling, etc.).
- [ ] **Contributing Guide** - Add `CONTRIBUTING.md` with setup, test, and PR process.

---

## Legend
- ✅ = Done (in this PR)
- [ ] = Pending