package database

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type TargetRepository struct {
	db *sql.DB
}

func NewTargetRepository(db *sql.DB) *TargetRepository {
	return &TargetRepository{db: db}
}

const targetColumns = "id, name, url, schedule, cert_expires_at, cert_notified_30d_at, cert_notified_10d_date, created_at, updated_at"

func scanTarget(scanner interface{ Scan(dest ...any) error }) (Target, error) {
	var t Target
	var cert sql.NullString
	var notified30d sql.NullString
	var notified10d sql.NullString
	if err := scanner.Scan(&t.ID, &t.Name, &t.URL, &t.Schedule, &cert, &notified30d, &notified10d, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return t, err
	}
	if cert.Valid {
		t.CertExpiresAt = &cert.String
	}
	if notified30d.Valid {
		t.CertNotified30dAt = &notified30d.String
	}
	if notified10d.Valid {
		t.CertNotified10dDate = &notified10d.String
	}
	return t, nil
}

func dbContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 2*time.Second)
}

func (r *TargetRepository) GetTargets() ([]Target, error) {
	return r.GetTargetsContext(context.Background())
}

func (r *TargetRepository) GetTargetsContext(ctx context.Context) ([]Target, error) {
	ctx, cancel := dbContext(ctx)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, "SELECT "+targetColumns+" FROM targets")
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
	return r.GetTargetByIDContext(context.Background(), id)
}

func (r *TargetRepository) GetTargetByIDContext(ctx context.Context, id int) (*Target, error) {
	ctx, cancel := dbContext(ctx)
	defer cancel()
	t, err := scanTarget(r.db.QueryRowContext(ctx, "SELECT "+targetColumns+" FROM targets WHERE id = ?", id))
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

// UpdateCertState atomically stores the observed TLS certificate expiry together
// with the notification bookkeeping for it. Passing nil for a notification
// column clears it.
func (r *TargetRepository) UpdateCertState(id int, expiresAt string, notified30dAt, notified10dDate *string) error {
	_, err := r.db.Exec(
		"UPDATE targets SET cert_expires_at = ?, cert_notified_30d_at = ?, cert_notified_10d_date = ? WHERE id = ?",
		expiresAt, notified30dAt, notified10dDate, id,
	)
	return err
}

// UpdateTarget updates the name and schedule of an existing target.
func (r *TargetRepository) UpdateTarget(id int, name string, schedule string) error {
	now := Now()
	res, err := r.db.Exec(
		"UPDATE targets SET name = ?, schedule = ?, updated_at = ? WHERE id = ?",
		name, schedule, &now, id,
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
		t.Name, t.URL, t.Schedule, &t.CreatedAt, &t.UpdatedAt,
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
		c.TargetID, c.StatusCode, c.ResponseTimeMS, isUp, c.ErrorMessage, &c.CheckedAt,
	)
	return err
}

func (r *TargetRepository) CreateIncident(inc *Incident) error {
	if inc.Type == "" {
		inc.Type = IncidentTypeGoingDown
	}
	if inc.StartedAt.IsZero() {
		inc.StartedAt = Now()
	}
	inc.Timestamp = inc.StartedAt
	inc.CreatedAt = Now()
	resolved := 0
	if inc.Resolved {
		resolved = 1
	}
	result, err := r.db.Exec(
		"INSERT INTO incidents (target_id, started_at, ended_at, cause, resolved, created_at, type, fingerprint, is_read) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		inc.TargetID, &inc.StartedAt, inc.EndedAt, inc.Cause, resolved, &inc.CreatedAt, inc.Type, inc.Fingerprint, boolToInt(inc.IsRead),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	inc.ID = int(id)
	return err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (r *TargetRepository) HasIncident(targetID int, incidentType string, fingerprint string) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM incidents WHERE target_id = ? AND type = ? AND fingerprint = ?", targetID, incidentType, fingerprint).Scan(&count)
	return count > 0, err
}

const incidentColumns = "i.id, i.target_id, i.started_at, i.ended_at, i.cause, i.resolved, i.created_at, i.type, i.fingerprint, i.is_read, t.name, t.url"

func scanIncident(scanner interface{ Scan(dest ...any) error }) (Incident, error) {
	var incident Incident
	var resolved, isRead int
	err := scanner.Scan(
		&incident.ID, &incident.TargetID, &incident.StartedAt, &incident.EndedAt, &incident.Cause,
		&resolved, &incident.CreatedAt, &incident.Type, &incident.Fingerprint, &isRead,
		&incident.TargetName, &incident.TargetURL,
	)
	incident.Resolved = resolved == 1
	incident.IsRead = isRead == 1
	incident.Timestamp = incident.StartedAt
	return incident, err
}

func (r *TargetRepository) GetIncidents() ([]Incident, error) {
	rows, err := r.db.Query("SELECT " + incidentColumns + " FROM incidents i JOIN targets t ON t.id = i.target_id ORDER BY i.started_at DESC, i.id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	incidents := make([]Incident, 0)
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}
	return incidents, rows.Err()
}

func (r *TargetRepository) MarkIncidentRead(id int) error {
	_, err := r.db.Exec("UPDATE incidents SET is_read = 1 WHERE id = ?", id)
	return err
}

func (r *TargetRepository) MarkAllIncidentsRead() error {
	_, err := r.db.Exec("UPDATE incidents SET is_read = 1 WHERE is_read = 0")
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
	return r.GetTargetsWithRecentChecksContext(context.Background(), limit)
}

func (r *TargetRepository) GetTargetsWithRecentChecksContext(ctx context.Context, limit int) ([]TargetWithChecks, error) {
	targets, err := r.GetTargetsContext(ctx)
	if err != nil {
		return nil, err
	}

	var result []TargetWithChecks
	for _, t := range targets {
		checks, err := r.GetRecentChecksByTargetIDContext(ctx, t.ID, limit)
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
	return r.GetRecentChecksByTargetIDContext(context.Background(), targetID, limit)
}

func (r *TargetRepository) GetRecentChecksByTargetIDContext(ctx context.Context, targetID int, limit int) ([]Check, error) {
	ctx, cancel := dbContext(ctx)
	defer cancel()
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, target_id, status_code, response_time_ms, is_up, error_message, checked_at FROM checks WHERE target_id = ? ORDER BY checked_at DESC, id DESC LIMIT ?",
		targetID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	var checks []Check
	for rows.Next() {
		var c Check
		var isUpInt int
		if err := rows.Scan(&c.ID, &c.TargetID, &c.StatusCode, &c.ResponseTimeMS, &isUpInt, &c.ErrorMessage, &c.CheckedAt); err != nil {
			return nil, err
		}
		c.IsUp = isUpInt == 1
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

// GetLastCheckByTargetID returns the most recent check for a target, or nil if
// the target has not been checked yet.
func (r *TargetRepository) GetLastCheckByTargetID(targetID int) (*Check, error) {
	return r.GetLastCheckByTargetIDContext(context.Background(), targetID)
}

func (r *TargetRepository) GetLastCheckByTargetIDContext(ctx context.Context, targetID int) (*Check, error) {
	checks, err := r.GetRecentChecksByTargetIDContext(ctx, targetID, 1)
	if err != nil {
		return nil, err
	}
	if len(checks) == 0 {
		return nil, nil
	}
	return &checks[0], nil
}
