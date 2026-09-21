package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	"uptime/internal/database"
)

func TestPollingServiceStopsCleanly(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "shutdown.db"))
	if err != nil {
		t.Fatal(err)
	}
	repo := database.NewTargetRepository(db)
	service := NewPollingService(repo, "")
	service.resyncInterval = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.Start(ctx)
		close(done)
	}()
	cancel()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()
	if err := service.Wait(waitCtx); err != nil {
		t.Fatalf("polling service did not stop: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start did not return after shutdown")
	}
	_ = db.Close()
}

func TestPollingServiceWaitsWhenStartCannotSync(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "shutdown-error.db"))
	if err != nil {
		t.Fatal(err)
	}
	repo := database.NewTargetRepository(db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	service := NewPollingService(repo, "")
	go service.Start(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Wait(ctx); err != nil {
		t.Fatalf("service did not finish after sync failure: %v", err)
	}
}
