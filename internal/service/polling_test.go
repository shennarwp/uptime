package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uptime/internal/database"
)

func TestPollingService_PingAndPoll(t *testing.T) {
	// Start test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "uptime_poll_test_*")
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
	err = repo.CreateTarget(&database.Target{
		Name:     "Polled Target",
		URL:      ts.URL,
		Schedule: "@every 1s",
	})
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	pollingSvc := NewPollingService(repo)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test pingTarget directly
	targets, err := repo.GetTargets()
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}

	pollingSvc.pingTarget(targets[0])

	checks, err := repo.GetRecentChecksByTargetID(targets[0].ID, 10)
	if err != nil {
		t.Fatalf("failed to get checks: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if !checks[0].IsUp {
		t.Errorf("expected check to be up")
	}

	// Test Start cron with canceled context
	go pollingSvc.Start(ctx)
	<-ctx.Done()
}

func TestPollingService_Resync(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uptime_poll_resync_*")
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
	target := &database.Target{
		Name:     "Resync Target",
		URL:      "http://example.com",
		Schedule: "@every 1h",
	}
	if err := repo.CreateTarget(target); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	pollingSvc := NewPollingService(repo)

	if err := pollingSvc.sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}
	if entry, ok := pollingSvc.entries[target.ID]; !ok || entry.schedule != "@every 1h" {
		t.Fatalf("expected entry for target %d to be registered with @every 1h", target.ID)
	}
	baseCount := len(pollingSvc.entries) - 1

	updater, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db for update: %v", err)
	}
	_, err = updater.Exec("UPDATE targets SET schedule = ? WHERE id = ?", "@every 5m", target.ID)
	updater.Close()
	if err != nil {
		t.Fatalf("failed to update schedule: %v", err)
	}

	if err := pollingSvc.sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}
	if len(pollingSvc.entries) != baseCount+1 {
		t.Fatalf("expected %d entries after reschedule, got %d", baseCount+1, len(pollingSvc.entries))
	}
	if entry := pollingSvc.entries[target.ID]; entry == nil || entry.schedule != "@every 5m" {
		t.Fatalf("expected rescheduled entry with @every 5m, got %+v", pollingSvc.entries[target.ID])
	}

	if err := repo.DeleteTarget(target.ID); err != nil {
		t.Fatalf("failed to delete target: %v", err)
	}
	if err := pollingSvc.sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}
	if len(pollingSvc.entries) != baseCount {
		t.Fatalf("expected %d entries after deletion, got %d", baseCount, len(pollingSvc.entries))
	}
}
