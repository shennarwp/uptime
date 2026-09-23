package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTargetRepositoryContextQueries(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	targets, err := repo.GetTargetsContext(ctx)
	if err != nil || len(targets) == 0 {
		t.Fatalf("GetTargetsContext failed: %v", err)
	}
	if _, err := repo.GetTargetByIDContext(ctx, targets[0].ID); err != nil {
		t.Fatalf("GetTargetByIDContext failed: %v", err)
	}
	if _, err := repo.GetRecentChecksByTargetIDContext(ctx, targets[0].ID, 5); err != nil {
		t.Fatalf("GetRecentChecksByTargetIDContext failed: %v", err)
	}
	if _, err := repo.GetLastCheckByTargetIDContext(ctx, targets[0].ID); err != nil {
		t.Fatalf("GetLastCheckByTargetIDContext failed: %v", err)
	}
	if _, err := repo.GetTargetsWithRecentChecksContext(ctx, 5); err != nil {
		t.Fatalf("GetTargetsWithRecentChecksContext failed: %v", err)
	}
}

func setupTestDB(t *testing.T) (*TargetRepository, func()) {
	tmpDir, err := os.MkdirTemp("", "uptime_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open test db: %v", err)
	}

	repo := NewTargetRepository(db)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestTargetRepository_CRUDAndChecksAndIncidents(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	initialCount := 0
	targets, err := repo.GetTargets()
	if err == nil {
		initialCount = len(targets)
	}

	// Test CreateTarget
	target := &Target{
		Name:     "Test Target Unique",
		URL:      "http://example.com/unique",
		Schedule: "@every 1m",
	}
	err = repo.CreateTarget(target)
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}
	if target.ID == 0 {
		t.Errorf("expected target ID to be set, got 0")
	}

	// Test GetTargets
	targets, err = repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}
	if len(targets) != initialCount+1 {
		t.Errorf("expected %d targets, got %d", initialCount+1, len(targets))
	}

	// Test GetTargetByID
	fetched, err := repo.GetTargetByID(target.ID)
	if err != nil {
		t.Fatalf("failed to get target by ID: %v", err)
	}
	if fetched.Name != target.Name {
		t.Errorf("expected name %s, got %s", target.Name, fetched.Name)
	}
	if fetched.CertExpiresAt != nil {
		t.Errorf("expected cert_expires_at to be nil for a new target, got %v", *fetched.CertExpiresAt)
	}

	// Test UpdateCertExpiresAt
	expiry := "2026-09-04T15:40:22Z"
	if err := repo.UpdateCertExpiresAt(target.ID, expiry); err != nil {
		t.Fatalf("failed to update cert expiry: %v", err)
	}
	fetched, err = repo.GetTargetByID(target.ID)
	if err != nil {
		t.Fatalf("failed to get target after cert expiry update: %v", err)
	}
	if fetched.CertExpiresAt == nil || *fetched.CertExpiresAt != expiry {
		t.Errorf("expected cert_expires_at %s, got %v", expiry, fetched.CertExpiresAt)
	}
	targets, err = repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets after cert expiry update: %v", err)
	}
	foundExpiry := false
	for _, t := range targets {
		if t.ID == target.ID && t.CertExpiresAt != nil && *t.CertExpiresAt == expiry {
			foundExpiry = true
			break
		}
	}
	if !foundExpiry {
		t.Errorf("expected cert_expires_at %s in GetTargets result", expiry)
	}

	// Test CreateCheck (Up and Down)
	statusCode := 200
	responseTime := 50
	err = repo.CreateCheck(&Check{
		TargetID:       target.ID,
		StatusCode:     &statusCode,
		ResponseTimeMS: &responseTime,
		IsUp:           true,
	})
	if err != nil {
		t.Fatalf("failed to create check: %v", err)
	}

	errMsg := "connection refused"
	err = repo.CreateCheck(&Check{
		TargetID:     target.ID,
		IsUp:         false,
		ErrorMessage: &errMsg,
	})
	if err != nil {
		t.Fatalf("failed to create down check: %v", err)
	}

	// Test GetRecentChecksByTargetID
	checks, err := repo.GetRecentChecksByTargetID(target.ID, 10)
	if err != nil {
		t.Fatalf("failed to get recent checks: %v", err)
	}
	if len(checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(checks))
	}

	// Test GetTargetsWithRecentChecks
	targetsWithChecks, err := repo.GetTargetsWithRecentChecks(5)
	if err != nil {
		t.Fatalf("failed to get targets with recent checks: %v", err)
	}
	foundCreated := false
	for _, twc := range targetsWithChecks {
		if twc.ID == target.ID {
			foundCreated = true
			if len(twc.Checks) != 2 {
				t.Errorf("expected 2 checks for created target, got %d", len(twc.Checks))
			}
		}
	}
	if !foundCreated {
		t.Errorf("created target not found in GetTargetsWithRecentChecks")
	}

	// Test Incident operations
	cause := "server down"
	inc := &Incident{
		TargetID:  target.ID,
		StartedAt: Now(),
		Cause:     &cause,
		Resolved:  false,
	}
	err = repo.CreateIncident(inc)
	if err != nil {
		t.Fatalf("failed to create incident: %v", err)
	}

	incidents, err := repo.GetIncidents()
	if err != nil {
		t.Fatalf("failed to get incidents: %v", err)
	}
	if len(incidents) != 1 || incidents[0].TargetName != target.Name || incidents[0].TargetURL != target.URL || incidents[0].IsRead {
		t.Fatalf("unexpected incident list: %+v", incidents)
	}
	fingerprint := "cert-expiry"
	certIncident := &Incident{TargetID: target.ID, Type: IncidentTypeCert30Days, Fingerprint: &fingerprint, Resolved: true, IsRead: true}
	if err := repo.CreateIncident(certIncident); err != nil {
		t.Fatalf("failed to create certificate incident: %v", err)
	}
	if err := repo.CreateIncident(&Incident{TargetID: 999999, Type: IncidentTypeGoingDown}); err == nil {
		t.Fatal("expected foreign-key error when creating incident for missing target")
	}
	exists, err := repo.HasIncident(target.ID, IncidentTypeCert30Days, fingerprint)
	if err != nil || !exists {
		t.Fatalf("expected certificate incident fingerprint to exist, got exists=%v err=%v", exists, err)
	}
	exists, err = repo.HasIncident(target.ID, IncidentTypeCert30Days, "different")
	if err != nil || exists {
		t.Fatalf("expected different fingerprint to be absent, got exists=%v err=%v", exists, err)
	}
	if err := repo.MarkIncidentRead(inc.ID); err != nil {
		t.Fatalf("failed to mark incident read: %v", err)
	}
	if err := repo.MarkAllIncidentsRead(); err != nil {
		t.Fatalf("failed to mark all incidents read: %v", err)
	}
	incidents, err = repo.GetIncidents()
	if err != nil || len(incidents) != 2 || !incidents[0].IsRead || !incidents[1].IsRead {
		t.Fatalf("expected all incidents to be read, got incidents=%+v err=%v", incidents, err)
	}

	// Test CloseIncident (Assuming ID 1 or getting incident ID)
	err = repo.CloseIncident(1)
	if err != nil {
		t.Fatalf("failed to close incident: %v", err)
	}

	// Test DeleteTarget
	err = repo.DeleteTarget(target.ID)
	if err != nil {
		t.Fatalf("failed to delete target: %v", err)
	}

	targetsAfterDelete, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets after delete: %v", err)
	}
	if len(targetsAfterDelete) != initialCount {
		t.Errorf("expected %d targets after deletion, got %d", initialCount, len(targetsAfterDelete))
	}
}
