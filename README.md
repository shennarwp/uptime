# Uptime Monitor

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

## Project Structure

- `cmd/uptime/` - Application entry point (`main.go`)
- `internal/database/` - SQLite connection, database models, migrations, and target repository
- `internal/service/` - Target and polling services
- `internal/handler/` - HTTP API handlers
- `web/` - React frontend (Vite, TypeScript, CSS)
  - `src/components/` - React components (`Sidebar`, `TargetCard`, `CheckHistoryBar`, `Header`, `TruncatedUrl`) and unit tests
