package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"uptime/internal/database"
	"uptime/internal/service"
)

func TestTargetHandler_GetTargets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uptime_handler_test_*")
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
		Name:     "Handler Target Unique",
		URL:      "http://example.com/handler",
		Schedule: "@every 1m",
	})
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	svc := service.NewTargetService(repo)
	handler := NewTargetHandler(svc)

	initialTargets, _ := repo.GetTargets()
	expectedCount := len(initialTargets)

	req := httptest.NewRequest("GET", "/api/targets", nil)
	rec := httptest.NewRecorder()

	handler.GetTargets(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", res.Header.Get("Content-Type"))
	}

	var targets []database.TargetWithChecks
	err = json.NewDecoder(res.Body).Decode(&targets)
	if err != nil {
		t.Fatalf("failed to decode response json: %v", err)
	}

	if len(targets) != expectedCount {
		t.Errorf("expected %d targets in response, got %d", expectedCount, len(targets))
	}

	foundHandlerTarget := false
	for _, target := range targets {
		if target.Name == "Handler Target Unique" {
			foundHandlerTarget = true
			break
		}
	}
	if !foundHandlerTarget {
		t.Errorf("expected to find 'Handler Target Unique' in targets response")
	}
}
