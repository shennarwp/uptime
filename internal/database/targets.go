package database

import (
	"database/sql"
	"log"
)

type TargetRepository struct {
	db *sql.DB
}

func NewTargetRepository(db *sql.DB) *TargetRepository {
	return &TargetRepository{db: db}
}

func (r *TargetRepository) GetTargets() ([]Target, error) {
	rows, err := r.db.Query("SELECT id, name, url, interval_s, created_at, updated_at FROM targets")
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	var targets []Target
	for rows.Next() {
		var t Target
		if err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.IntervalS, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func (r *TargetRepository) GetTargetByID(id int) (*Target, error) {
	var t Target
	err := r.db.QueryRow(
		"SELECT id, name, url, interval_s, created_at, updated_at FROM targets WHERE id = ?", id,
	).Scan(&t.ID, &t.Name, &t.URL, &t.IntervalS, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TargetRepository) CreateTarget(t *Target) error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	result, err := r.db.Exec(
		"INSERT INTO targets (name, url, interval_s, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		t.Name, t.URL, t.IntervalS, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = int(id)
	return nil
}

func (r *TargetRepository) DeleteTarget(id int) error {
	_, err := r.db.Exec("DELETE FROM targets WHERE id = ?", id)
	return err
}

func (r *TargetRepository) CreateCheck(c *Check) error {
	c.CheckedAt = Now()
	isUp := 0
	if c.IsUp {
		isUp = 1
	}
	_, err := r.db.Exec(
		"INSERT INTO checks (target_id, status_code, response_time_ms, is_up, error_message, checked_at) VALUES (?, ?, ?, ?, ?, ?)",
		c.TargetID, c.StatusCode, c.ResponseTimeMS, isUp, c.ErrorMessage, c.CheckedAt,
	)
	return err
}

func (r *TargetRepository) CreateIncident(inc *Incident) error {
	inc.CreatedAt = Now()
	resolved := 0
	if inc.Resolved {
		resolved = 1
	}
	_, err := r.db.Exec(
		"INSERT INTO incidents (target_id, started_at, ended_at, cause, resolved, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		inc.TargetID, inc.StartedAt, inc.EndedAt, inc.Cause, resolved, inc.CreatedAt,
	)
	return err
}

func (r *TargetRepository) CloseIncident(id int) error {
	now := Now()
	_, err := r.db.Exec(
		"UPDATE incidents SET ended_at = ?, resolved = 1 WHERE id = ? AND resolved = 0",
		&now, id,
	)
	return err
}
