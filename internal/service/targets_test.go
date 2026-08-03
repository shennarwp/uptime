package service

import (
	"os"
	"path/filepath"
	"testing"
	"uptime/internal/database"
)

func TestTargetService(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uptime_svc_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewTargetRepository(db)
	svc := NewTargetService(repo)

	initialTargets, _ := repo.GetTargets()
	initialCount := len(initialTargets)

	// Create test target
	err = repo.CreateTarget(&database.Target{
		Name:     "Svc Test Unique",
		URL:      "http://localhost/svc",
		Schedule: "@every 1m",
	})
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	targets, err := svc.GetTargets()
	if err != nil {
		t.Fatalf("svc.GetTargets error: %v", err)
	}
	if len(targets) != initialCount+1 {
		t.Errorf("expected %d targets, got %d", initialCount+1, len(targets))
	}

	twc, err := svc.GetTargetsWithRecentChecks(10)
	if err != nil {
		t.Fatalf("svc.GetTargetsWithRecentChecks error: %v", err)
	}
	if len(twc) != initialCount+1 {
		t.Errorf("expected %d targets with checks, got %d", initialCount+1, len(twc))
	}
}
