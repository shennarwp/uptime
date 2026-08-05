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

const targetColumns = "id, name, url, schedule, cert_expires_at, created_at, updated_at"

func scanTarget(scanner interface{ Scan(dest ...any) error }) (Target, error) {
	var t Target
	var cert sql.NullString
	if err := scanner.Scan(&t.ID, &t.Name, &t.URL, &t.Schedule, &cert, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return t, err
	}
	if cert.Valid {
		t.CertExpiresAt = &cert.String
	}
	return t, nil
}

func (r *TargetRepository) GetTargets() ([]Target, error) {
	rows, err := r.db.Query("SELECT " + targetColumns + " FROM targets")
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
		t, err := scanTarget(rows)
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func (r *TargetRepository) GetTargetByID(id int) (*Target, error) {
	t, err := scanTarget(r.db.QueryRow("SELECT "+targetColumns+" FROM targets WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateCertExpiresAt stores the TLS certificate expiry observed for a target.
func (r *TargetRepository) UpdateCertExpiresAt(id int, expiresAt string) error {
	_, err := r.db.Exec("UPDATE targets SET cert_expires_at = ? WHERE id = ?", expiresAt, id)
	return err
}

// UpdateTarget updates the name and schedule of an existing target.
func (r *TargetRepository) UpdateTarget(id int, name string, schedule string) error {
	now := Now()
	res, err := r.db.Exec(
		"UPDATE targets SET name = ?, schedule = ?, updated_at = ? WHERE id = ?",
		name, schedule, now, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *TargetRepository) CreateTarget(t *Target) error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	result, err := r.db.Exec(
		"INSERT INTO targets (name, url, schedule, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		t.Name, t.URL, t.Schedule, t.CreatedAt, t.UpdatedAt,
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

func (r *TargetRepository) GetTargetsWithRecentChecks(limit int) ([]TargetWithChecks, error) {
	targets, err := r.GetTargets()
	if err != nil {
		return nil, err
	}

	var result []TargetWithChecks
	for _, t := range targets {
		checks, err := r.GetRecentChecksByTargetID(t.ID, limit)
		if err != nil {
			return nil, err
		}
		result = append(result, TargetWithChecks{
			Target: t,
			Checks: checks,
		})
	}
	return result, nil
}

func (r *TargetRepository) GetRecentChecksByTargetID(targetID int, limit int) ([]Check, error) {
	rows, err := r.db.Query(
		"SELECT id, target_id, status_code, response_time_ms, is_up, error_message, checked_at FROM checks WHERE target_id = ? ORDER BY checked_at DESC, id DESC LIMIT ?",
		targetID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		var isUpInt int
		if err := rows.Scan(&c.ID, &c.TargetID, &c.StatusCode, &c.ResponseTimeMS, &isUpInt, &c.ErrorMessage, &c.CheckedAt); err != nil {
			return nil, err
		}
		c.IsUp = (isUpInt == 1)
		checks = append(checks, c)
	}
	return checks, rows.Err()
}
