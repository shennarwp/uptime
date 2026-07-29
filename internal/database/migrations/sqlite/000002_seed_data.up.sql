INSERT OR IGNORE INTO targets (id, name, url, interval_s) VALUES
    (1, 'Example',  'https://example.com',  30),
    (2, 'Google',   'https://google.com',   60),
    (3, 'GitHub',   'https://github.com',   120);

INSERT OR IGNORE INTO checks (id, target_id, status_code, response_time_ms, is_up, checked_at) VALUES
    (1, 1, 200,  120, 1, datetime('now', '-2 minutes')),
    (2, 1, 200,  150, 1, datetime('now', '-1 minute')),
    (3, 1, 200,  110, 1, datetime('now')),
    (4, 2, 200,  45,  1, datetime('now', '-2 minutes')),
    (5, 2, 200,  52,  1, datetime('now', '-1 minute')),
    (6, 2, 503,  null, 0, datetime('now')),
    (7, 3, 200,  230, 1, datetime('now', '-2 minutes')),
    (8, 3, 200,  200, 1, datetime('now', '-1 minute')),
    (9, 3, 200,  190, 1, datetime('now'));
