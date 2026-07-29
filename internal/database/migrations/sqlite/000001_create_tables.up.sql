CREATE TABLE IF NOT EXISTS targets (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    url         TEXT    NOT NULL,
    interval_s  INTEGER NOT NULL DEFAULT 60,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS checks (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id        INTEGER NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    status_code      INTEGER,
    response_time_ms INTEGER,
    is_up            INTEGER NOT NULL,
    error_message    TEXT,
    checked_at       TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_checks_target_checked ON checks (target_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_checks_target_failing ON checks (target_id, is_up) WHERE NOT is_up;

CREATE TABLE IF NOT EXISTS incidents (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id   INTEGER NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    started_at  TEXT    NOT NULL,
    ended_at    TEXT,
    cause       TEXT,
    resolved    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_incidents_target ON incidents (target_id, started_at DESC);
