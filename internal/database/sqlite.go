package database

import (
    "database/sql"
    "embed"
    "fmt"
    "path"
    "sort"
    "strings"

    _ "modernc.org/sqlite"
)

//go:embed migrations/sqlite/*.sql
var migrations embed.FS

func Open(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, fmt.Errorf("open: %w", err)
    }
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("ping: %w", err)
    }
    pragmas := []string{
        "PRAGMA journal_mode = WAL",
        "PRAGMA foreign_keys = ON",
    }
    for _, p := range pragmas {
        if _, err := db.Exec(p); err != nil {
            return nil, fmt.Errorf("%s: %w", p, err)
        }
    }
    if err := migrate(db); err != nil {
        return nil, fmt.Errorf("migrate: %w", err)
    }
    return db, nil
}

func migrate(db *sql.DB) error {
    db.Exec("CREATE TABLE IF NOT EXISTS _migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))")

    entries, err := migrations.ReadDir("migrations/sqlite")
    if err != nil {
        return err
    }

    var files []string
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
            files = append(files, e.Name())
        }
    }
    sort.Strings(files)

    for _, f := range files {
        version := strings.TrimSuffix(f, ".up.sql")
        var count int
        db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE version = ?", version).Scan(&count)
        if count > 0 {
            continue
        }
        up, err := migrations.ReadFile(path.Join("migrations/sqlite", f))
        if err != nil {
            return err
        }
        tx, err := db.Begin()
        if err != nil {
            return err
        }
        if _, err := tx.Exec(string(up)); err != nil {
            tx.Rollback()
            return fmt.Errorf("%s: %w", f, err)
        }
        if _, err := tx.Exec("INSERT INTO _migrations (version) VALUES (?)", version); err != nil {
            tx.Rollback()
            return err
        }
        if err := tx.Commit(); err != nil {
            return err
        }
    }
    return nil
}
