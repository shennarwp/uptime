package database

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type Timestamp struct {
	time.Time
}

func (t *Timestamp) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		parsed, err := time.Parse("2006-01-02 15:04:05", v)
		if err != nil {
			return fmt.Errorf("scan timestamp: %w", err)
		}
		t.Time = parsed
		return nil
	case []byte:
		return t.Scan(string(v))
	default:
		return fmt.Errorf("unsupported scan type for Timestamp: %T", src)
	}
}

func (t Timestamp) Value() (driver.Value, error) {
	return t.Time.UTC().Format("2006-01-02 15:04:05"), nil
}

func Now() Timestamp {
	return Timestamp{Time: time.Now().UTC()}
}

// Target represents a monitored URL endpoint.
type Target struct {
	// ID is the unique identifier of the target.
	ID int `json:"id"`
	// Name is a human-readable label for the target.
	Name string `json:"name"`
	// URL is the endpoint that is being monitored.
	URL string `json:"url"`
	// Schedule is the cron expression used for polling.
	Schedule string `json:"schedule"`
	// CertExpiresAt is the NotAfter time of the TLS certificate most recently
	// observed for the target, in RFC3339 (UTC). Nil for HTTP targets or before
	// the first check observes a certificate.
	CertExpiresAt *string `json:"cert_expires_at,omitempty"`
	// CreatedAt is when the target was created.
	CreatedAt Timestamp `json:"created_at" swaggertype:"primitive,string"`
	// UpdatedAt is when the target was last updated.
	UpdatedAt Timestamp `json:"updated_at" swaggertype:"primitive,string"`
}

// TargetWithChecks is a target together with its most recent checks.
type TargetWithChecks struct {
	Target
	// Checks contains the recent health check results for the target.
	Checks []Check `json:"checks"`
}

// Check is a single health check result for a target.
type Check struct {
	// ID is the unique identifier of the check.
	ID int64 `json:"id"`
	// TargetID references the target this check belongs to.
	TargetID int `json:"target_id"`
	// StatusCode is the HTTP status code returned by the endpoint, if any.
	StatusCode *int `json:"status_code,omitempty"`
	// ResponseTimeMS is the response time in milliseconds, if any.
	ResponseTimeMS *int `json:"response_time_ms,omitempty"`
	// IsUp indicates whether the target was reachable.
	IsUp bool `json:"is_up"`
	// ErrorMessage contains the error message when the check failed, if any.
	ErrorMessage *string `json:"error_message,omitempty"`
	// CheckedAt is when the check was performed.
	CheckedAt Timestamp `json:"checked_at" swaggertype:"primitive,string"`
}

// Incident represents a period during which a target was down.
type Incident struct {
	// ID is the unique identifier of the incident.
	ID int `json:"id"`
	// TargetID references the target this incident belongs to.
	TargetID int `json:"target_id"`
	// StartedAt is when the incident started.
	StartedAt Timestamp `json:"started_at" swaggertype:"primitive,string"`
	// EndedAt is when the incident ended, if it has been resolved.
	EndedAt *Timestamp `json:"ended_at,omitempty" swaggertype:"primitive,string"`
	// Cause describes the reason for the incident, if known.
	Cause *string `json:"cause,omitempty"`
	// Resolved indicates whether the incident has been resolved.
	Resolved bool `json:"resolved"`
	// CreatedAt is when the incident record was created.
	CreatedAt Timestamp `json:"created_at" swaggertype:"primitive,string"`
}
