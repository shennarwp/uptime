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
    return t.Time.Format("2006-01-02 15:04:05"), nil
}

func Now() Timestamp {
    return Timestamp{Time: time.Now()}
}

type Target struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    URL       string    `json:"url"`
    IntervalS int       `json:"interval_s"`
    CreatedAt Timestamp `json:"created_at"`
    UpdatedAt Timestamp `json:"updated_at"`
}

type Check struct {
    ID             int64      `json:"id"`
    TargetID       int        `json:"target_id"`
    StatusCode     *int       `json:"status_code,omitempty"`
    ResponseTimeMS *int       `json:"response_time_ms,omitempty"`
    IsUp           bool       `json:"is_up"`
    ErrorMessage   *string    `json:"error_message,omitempty"`
    CheckedAt      Timestamp  `json:"checked_at"`
}

type Incident struct {
    ID        int        `json:"id"`
    TargetID  int        `json:"target_id"`
    StartedAt Timestamp  `json:"started_at"`
    EndedAt   *Timestamp `json:"ended_at,omitempty"`
    Cause     *string    `json:"cause,omitempty"`
    Resolved  bool       `json:"resolved"`
    CreatedAt Timestamp  `json:"created_at"`
}
