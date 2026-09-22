ALTER TABLE incidents ADD COLUMN type TEXT NOT NULL DEFAULT 'going_down';
ALTER TABLE incidents ADD COLUMN fingerprint TEXT;
ALTER TABLE incidents ADD COLUMN is_read INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_incidents_unread ON incidents (is_read, started_at DESC);
