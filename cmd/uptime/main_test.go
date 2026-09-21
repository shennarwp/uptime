package main

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"uptime/internal/database"
	"uptime/internal/service"
)

func TestShutdownStopsPollingBeforeServer(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "shutdown.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	polling := service.NewPollingService(database.NewTargetRepository(db), "")
	ctx, cancel := context.WithCancel(context.Background())
	go polling.Start(ctx)
	if err := shutdown(context.Background(), cancel, polling, &http.Server{}); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

func TestShutdownContinuesWhenPollingDoesNotStopBeforeDeadline(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "shutdown-timeout.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	polling := service.NewPollingService(database.NewTargetRepository(db), "")
	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()

	if err := shutdown(parent, func() {}, polling, &http.Server{}); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}
