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
