package database

import (
	"testing"
	"time"
)

func TestTimestamp_ScanAndValue(t *testing.T) {
	var ts Timestamp

	// Test string scan
	err := ts.Scan("2026-06-05 12:30:45")
	if err != nil {
		t.Fatalf("unexpected error scanning string: %v", err)
	}
	if ts.Time.Year() != 2026 || ts.Time.Hour() != 12 {
		t.Errorf("parsed time incorrect: %v", ts.Time)
	}

	// Test Value
	val, err := ts.Value()
	if err != nil {
		t.Fatalf("unexpected error getting value: %v", err)
	}
	if val != "2026-06-05 12:30:45" {
		t.Errorf("expected '2026-06-05 12:30:45', got %v", val)
	}

	// Test time.Time scan
	now := time.Now()
	err = ts.Scan(now)
	if err != nil {
		t.Fatalf("unexpected error scanning time.Time: %v", err)
	}

	// Test []byte scan
	err = ts.Scan([]byte("2026-01-01 00:00:00"))
	if err != nil {
		t.Fatalf("unexpected error scanning []byte: %v", err)
	}

	// Test invalid scan type
	err = ts.Scan(123)
	if err == nil {
		t.Errorf("expected error for unsupported scan type, got nil")
	}

	// Test invalid string format
	err = ts.Scan("invalid-date")
	if err == nil {
		t.Errorf("expected error for invalid date string, got nil")
	}
}

func TestNow(t *testing.T) {
	n := Now()
	if n.Time.IsZero() {
		t.Errorf("Now() returned zero time")
	}
}
