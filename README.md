# <img src="web/public/logo.svg" alt="" width="40" height="40" /> Uptime Monitor

A lightweight, self-hosted uptime monitoring application featuring a Go backend (SQLite) and a modern React frontend (TypeScript & Vite).

## Features

- **Automated Polling:** Background health checks scheduled per target using cron expressions with response time tracking and status code verification.
- **SQLite Storage:** Built-in SQLite database with automated migration management.
- **Modern Web Dashboard:**
  - Responsive layout optimized for both desktop and mobile views.
  - Compact check history bars with recent status check indicators.
  - Collapsible sidebar navigation for mobile devices with indicator arrows.
  - URL truncation and expansion on hover.
- **Comprehensive Testing:**
  - Go unit tests for database models, repositories, polling service, and HTTP handlers.
  - Vitest & React Testing Library component tests for the frontend.

---

## Getting Started

### Prerequisites

- **Go** (v1.26 or newer)
- **Node.js** (v18 or newer recommended) & npm

---

### Running the Backend (Go)

1. Navigate to the project root directory.
2. Run the application:
   ```bash
   go run cmd/uptime/main.go
   ```
   The backend server will start on `:8080`.

3. Run Go tests:
   ```bash
   go test -v ./...
   ```

---

### Running the Frontend (React / Vite)

1. Navigate to the `web` directory:
   ```bash
   cd web
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Run the development server (proxies `/api` to `:8080`):
   ```bash
   npm run dev
   ```
   Dashboard is available on `http://localhost:5173/`.
4. Run frontend tests:
   ```bash
   npm test
   ```
5. Build for production:
   ```bash
   npm run build
   ```

---

### Docker

Build the image (multi-stage; uses `nginx:alpine` as the runtime base):

```bash
docker build -f Dockerfile-alpine -t shennarwp/uptime:alpine-latest .
```

Run the container. The database is persisted in a mounted directory (see `UPTIME_DB_PATH` below), so it survives container restarts and removal:

```bash
docker run --detach \
  --name uptime \
  --restart always \
  -v ~/uptime/data:/app/data \
  --expose 80 \
  shennarwp/uptime:alpine-latest
```

Notes:
- The image exposes port `80` (nginx serving the React frontend and proxying `/api` to the Go backend on `:8080`).
- `UPTIME_DB_PATH` defaults to `/app/data/uptime.db`; the SQLite database (including its WAL files) lives in the mounted volume, so mount a directory (not a single file) or data will be lost on container recreation.
- Override the database path at runtime with `-e UPTIME_DB_PATH=/some/other/path.db`.

### Protecting mutating endpoints

Write endpoints (`PUT /api/target/{id}`) are guarded by a bearer token read from the `UPTIME_API_TOKEN` environment variable. Requests must send an `Authorization: Bearer <token>` header; anything else returns `401 Unauthorized`.

```bash
docker run ... -e UPTIME_API_TOKEN=your-secret-token ...
```

If `UPTIME_API_TOKEN` is unset the guard **fails closed** — all writes are denied (a warning is logged at startup). You must set the variable for the edit/login flow to work.

The web UI has a **Login** button in the header. Entering a token calls `POST /api/auth/verify` to confirm it's correct; on success the token is stored in the browser's `localStorage` and the button becomes **Logout**. The edit buttons on target cards are only shown while logged in, and the token is cleared automatically if a save is ever rejected with 401.

---

## API Documentation

The backend exposes an OpenAPI (Swagger) specification generated from Go annotations using [swaggo/swag](https://github.com/swaggo/swag).

- **Spec files:** `api/swagger.json` and `api/swagger.yaml`
- **Interactive UI:** served by the backend at `http://localhost:8080/swagger/` (Swagger UI) when running the Go server.

### Regenerating the spec

The spec is generated from `swagger` comment annotations in the code (`cmd/uptime/main.go`, `internal/handler/`, `internal/database/models.go`). After changing the API, regenerate with:

```bash
go tool swag init -g cmd/uptime/main.go --output api
```

Commit the regenerated `api/` files together with your API changes.

---

## Project Structure

- `cmd/uptime/` - Application entry point (`main.go`)
- `api/` - Generated OpenAPI spec (`swagger.json`, `swagger.yaml`, `docs.go`)
- `internal/database/` - SQLite connection, database models, migrations, and target repository
- `internal/service/` - Target and polling services
- `internal/handler/` - HTTP API handlers
- `web/` - React frontend (Vite, TypeScript, CSS)
  - `src/components/` - React components (`Sidebar`, `TargetCard`, `CheckHistoryBar`, `Header`, `TruncatedUrl`) and unit tests
