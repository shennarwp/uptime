package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func TestTargetHandler_CreateTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uptime_handler_create_*")
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
	svc := service.NewTargetService(repo)
	handler := NewTargetHandler(svc)

	before, _ := repo.GetTargets()
	beforeCount := len(before)

	// Valid create -> 201
	req := httptest.NewRequest("POST", "/api/targets", strings.NewReader(`{"name":"New Target","url":"https://example.com/api","schedule":"0 0 */3 * * *"}`))
	rec := httptest.NewRecorder()
	handler.CreateTarget(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var created database.Target
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response json: %v", err)
	}
	if created.Name != "New Target" || created.URL != "https://example.com/api" || created.Schedule != "0 0 */3 * * *" {
		t.Errorf("unexpected created target: %+v", created)
	}
	if created.ID == 0 {
		t.Error("expected created target to have a nonzero id")
	}

	after, _ := repo.GetTargets()
	if len(after) != beforeCount+1 {
		t.Errorf("expected %d targets after create, got %d", beforeCount+1, len(after))
	}

	// Invalid body -> 400
	req = httptest.NewRequest("POST", "/api/targets", strings.NewReader(`not-json`))
	rec = httptest.NewRecorder()
	handler.CreateTarget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", rec.Code)
	}

	// Empty name -> 400
	req = httptest.NewRequest("POST", "/api/targets", strings.NewReader(`{"name":"  ","url":"https://example.com","schedule":"0 * * * * *"}`))
	rec = httptest.NewRecorder()
	handler.CreateTarget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", rec.Code)
	}

	// Invalid url -> 400
	for _, badURL := range []string{"", "example.com", "ftp://example.com", "https://"} {
		body := `{"name":"X","url":"` + badURL + `","schedule":"0 * * * * *"}`
		req = httptest.NewRequest("POST", "/api/targets", strings.NewReader(body))
		rec = httptest.NewRecorder()
		handler.CreateTarget(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for url %q, got %d", badURL, rec.Code)
		}
	}

	// Invalid schedule -> 400
	req = httptest.NewRequest("POST", "/api/targets", strings.NewReader(`{"name":"X","url":"https://example.com","schedule":"not-a-cron"}`))
	rec = httptest.NewRecorder()
	handler.CreateTarget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid schedule, got %d", rec.Code)
	}
}

func TestTargetHandler_UpdateTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uptime_handler_update_*")
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
		Name:     "Old Name",
		URL:      "http://example.com",
		Schedule: "0 * * * * *",
	}
	if err := repo.CreateTarget(target); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	svc := service.NewTargetService(repo)
	handler := NewTargetHandler(svc)

	req := httptest.NewRequest("PUT", "/api/target/"+strconv.Itoa(target.ID), strings.NewReader(`{"name":"New Name","schedule":"0 0 */3 * * *"}`))
	req.SetPathValue("id", strconv.Itoa(target.ID))
	rec := httptest.NewRecorder()

	handler.UpdateTarget(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var updated database.Target
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if updated.Name != "New Name" || updated.Schedule != "0 0 */3 * * *" {
		t.Errorf("expected updated name and schedule, got %+v", updated)
	}

	// Nonexistent target -> 404
	req = httptest.NewRequest("PUT", "/api/target/99999", strings.NewReader(`{"name":"X","schedule":"0 * * * * *"}`))
	req.SetPathValue("id", "99999")
	rec = httptest.NewRecorder()
	handler.UpdateTarget(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing target, got %d", rec.Code)
	}

	// Empty name -> 400
	req = httptest.NewRequest("PUT", "/api/target/"+strconv.Itoa(target.ID), strings.NewReader(`{"name":"  ","schedule":"0 * * * * *"}`))
	req.SetPathValue("id", strconv.Itoa(target.ID))
	rec = httptest.NewRecorder()
	handler.UpdateTarget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", rec.Code)
	}

	// Name too long -> 400
	longName := strings.Repeat("a", maxNameLength+1)
	req = httptest.NewRequest("PUT", "/api/target/"+strconv.Itoa(target.ID), strings.NewReader(`{"name":"`+longName+`","schedule":"0 * * * * *"}`))
	req.SetPathValue("id", strconv.Itoa(target.ID))
	rec = httptest.NewRecorder()
	handler.UpdateTarget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for overly long name, got %d", rec.Code)
	}

	// Name with control character -> 400
	req = httptest.NewRequest("PUT", "/api/target/"+strconv.Itoa(target.ID), strings.NewReader(`{"name":"Bad\nName","schedule":"0 * * * * *"}`))
	req.SetPathValue("id", strconv.Itoa(target.ID))
	rec = httptest.NewRecorder()
	handler.UpdateTarget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for control characters in name, got %d", rec.Code)
	}

	// Invalid schedule -> 400
	for _, bad := range []string{"not-a-cron", "0 * * * *", "* * * * * * *"} {
		req = httptest.NewRequest("PUT", "/api/target/"+strconv.Itoa(target.ID), strings.NewReader(`{"name":"X","schedule":"`+bad+`"}`))
		req.SetPathValue("id", strconv.Itoa(target.ID))
		rec = httptest.NewRecorder()
		handler.UpdateTarget(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid schedule %q, got %d", bad, rec.Code)
		}
	}

	// Valid 6-field cron spec -> 200
	req = httptest.NewRequest("PUT", "/api/target/"+strconv.Itoa(target.ID), strings.NewReader(`{"name":"Six Field","schedule":"*/5 * * * * *"}`))
	req.SetPathValue("id", strconv.Itoa(target.ID))
	rec = httptest.NewRecorder()
	handler.UpdateTarget(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for 6-field schedule, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	// Valid schedule with descriptor + unicode/special-char name -> 200
	req = httptest.NewRequest("PUT", "/api/target/"+strconv.Itoa(target.ID), strings.NewReader(`{"name":"Östlich & co (prod)#1","schedule":"@every 5m"}`))
	req.SetPathValue("id", strconv.Itoa(target.ID))
	rec = httptest.NewRecorder()
	handler.UpdateTarget(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid name/schedule, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
