# FIXME: Recommended Improvements

This document tracks recommended improvements for the Uptime Monitor project. Items are grouped by area and prioritized.

---

## Backend / API

### High Priority
- [ ] **Pagination & Filtering** - `GET /api/targets` returns all targets (up to 1500 checks each). Add `?limit=&offset=` and `?status=up|down` query params.
- [ ] **Structured Logging** - Replace `log.Printf` with JSON logger (zerolog/zap) in `internal/service/polling.go` and handlers for observability.

### Medium Priority
- [ ] **CORS Configuration** - Add CORS middleware if API consumed by external frontends.
- [ ] **Rate Limiting** - Add middleware for auth/abuse protection on mutating endpoints.
- [ ] **Config File Support** - Move from env-only to config file (YAML/TOML) for complex deployments.
- [ ] **Request Validation Middleware** - Move request validation from handler methods into reusable middleware or request-schema validation. The underlying field validators are already shared in `internal/handler/validation.go`.

### Low Priority
- [ ] **Metrics Endpoint** - Add `/metrics` for Prometheus scraping (request latency, check counts, etc.).
- [ ] **Webhook Support** - Generic webhook notifications in addition to ntfy.

---

## Frontend

### High Priority

### Medium Priority

### Low Priority
- [ ] **Bulk Operations** - Multi-select for bulk delete/enable/disable.
- [ ] **Export/Import** - JSON/YAML export of targets for backup/migration.

---

## DevOps / Infrastructure

### High Priority

### Medium Priority
- [ ] **Dependency Scanning** - Add `govulncheck` and `npm audit` to CI pipeline.
- [ ] **SBOM Generation** - Generate Software Bill of Materials in CI.
- [ ] **Automated Releases** - Semantic release on merge to main.

### Low Priority
- [ ] **Helm Chart** - Package for Kubernetes deployment.

---

## Testing

### High Priority

### Medium Priority
- [ ] **E2E Tests** - Playwright/Cypress for critical user flows (login, add target, edit, delete).
- [ ] **Load Testing** - k6 scripts for polling service under load.

---

## Documentation

- [ ] **Architecture Decision Records (ADRs)** - Document key decisions (SQLite, cron scheduling, etc.).
- [ ] **Contributing Guide** - Add `CONTRIBUTING.md` with setup, test, and PR process.

---
