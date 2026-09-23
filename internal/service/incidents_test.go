package service

import (
	"os"
	"path/filepath"
	"testing"
	"uptime/internal/database"
)

func databaseTestRepo(t *testing.T) (*database.TargetRepository, func()) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "service.db"))
	if err != nil {
		t.Fatal(err)
	}
	return database.NewTargetRepository(db), func() {
		_ = db.Close()
		_ = os.RemoveAll(dir)
	}
}

func TestIncidentService_ReadOperations(t *testing.T) {
	repo, cleanup := databaseTestRepo(t)
	defer cleanup()
	target := &database.Target{Name: "Service target", URL: "https://example.com", Schedule: "@every 1m"}
	if err := repo.CreateTarget(target); err != nil {
		t.Fatal(err)
	}
	incident := &database.Incident{TargetID: target.ID, Type: database.IncidentTypeGoingDown}
	if err := repo.CreateIncident(incident); err != nil {
		t.Fatal(err)
	}

	svc := NewIncidentService(repo)
	incidents, err := svc.GetIncidents()
	if err != nil || len(incidents) != 1 {
		t.Fatalf("GetIncidents returned %d incidents, err=%v", len(incidents), err)
	}
	if err := svc.MarkRead(incident.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkAllRead(); err != nil {
		t.Fatal(err)
	}
}
