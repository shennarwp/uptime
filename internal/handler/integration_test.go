package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"uptime/internal/database"
	"uptime/internal/service"
)

func integrationRouter(t *testing.T) (*sql.DB, http.Handler) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "integration.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Setenv("UPTIME_API_TOKEN", "integration-token")
	repo := database.NewTargetRepository(db)
	return db, NewRouter(
		NewTargetHandler(service.NewTargetService(repo)),
		service.NewEventBroker(),
		NewIncidentHandler(service.NewIncidentService(repo)),
	)
}

func TestAPIIntegrationCreateUpdateDelete(t *testing.T) {
	_, router := integrationRouter(t)

	create := httptest.NewRequest(http.MethodPost, "/api/v1/targets", bytes.NewBufferString(`{"name":"integration","url":"https://example.com","schedule":"0 * * * * *"}`))
	create.Header.Set("Authorization", "Bearer integration-token")
	created := httptest.NewRecorder()
	router.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", created.Code, created.Body.String())
	}
	var target database.Target
	if err := json.NewDecoder(created.Body).Decode(&target); err != nil {
		t.Fatal(err)
	}
	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/targets", nil))
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"name":"integration"`)) {
		t.Fatalf("list: unexpected response %d: %s", list.Code, list.Body.String())
	}

	update := httptest.NewRequest(http.MethodPut, "/api/v1/target/"+strconv.Itoa(target.ID), bytes.NewBufferString(`{"name":"updated","schedule":"0 0 * * * *"}`))
	update.Header.Set("Authorization", "Bearer integration-token")
	updated := httptest.NewRecorder()
	router.ServeHTTP(updated, update)
	if updated.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", updated.Code, updated.Body.String())
	}

	remove := httptest.NewRequest(http.MethodDelete, "/api/v1/target/"+strconv.Itoa(target.ID), nil)
	remove.Header.Set("Authorization", "Bearer integration-token")
	deleted := httptest.NewRecorder()
	router.ServeHTTP(deleted, remove)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", deleted.Code, deleted.Body.String())
	}

	list = httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/targets", nil))
	if list.Code != http.StatusOK || bytes.Contains(list.Body.Bytes(), []byte(`"name":"updated"`)) {
		t.Fatalf("list after delete: unexpected response %d: %s", list.Code, list.Body.String())
	}
}

func TestVersionedAndLegacyRoutes(t *testing.T) {
	_, router := integrationRouter(t)
	for _, path := range []string{"/api/targets", "/api/v1/targets"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", path, response.Code)
		}
	}
}

func TestIncidentAPIReadState(t *testing.T) {
	db, router := integrationRouter(t)
	result, err := db.Exec("INSERT INTO targets (name, url, schedule) VALUES ('incident target', 'https://example.com', '@every 1m')")
	if err != nil {
		t.Fatal(err)
	}
	targetID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO incidents (target_id, started_at, cause, type) VALUES (?, datetime('now'), 'went down', 'going_down')", targetID)
	if err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil))
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"is_read":false`)) {
		t.Fatalf("list: unexpected response %d: %s", list.Code, list.Body.String())
	}

	mark := httptest.NewRequest(http.MethodPatch, "/api/v1/incident/1/read", nil)
	mark.Header.Set("Authorization", "Bearer integration-token")
	marked := httptest.NewRecorder()
	router.ServeHTTP(marked, mark)
	if marked.Code != http.StatusNoContent {
		t.Fatalf("mark: expected 204, got %d: %s", marked.Code, marked.Body.String())
	}
	var isRead int
	if err := db.QueryRow("SELECT is_read FROM incidents WHERE id = 1").Scan(&isRead); err != nil {
		t.Fatal(err)
	}
	if isRead != 1 {
		t.Fatalf("expected incident to be marked read")
	}
}

func TestLatestIncidentAPIRequiresAuthAndReturnsIncidentID(t *testing.T) {
	db, router := integrationRouter(t)
	result, err := db.Exec("INSERT INTO targets (name, url, schedule) VALUES ('latest target', 'https://latest.example', '@every 1m')")
	if err != nil {
		t.Fatal(err)
	}
	targetID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	result, err = db.Exec(
		"INSERT INTO incidents (target_id, started_at, cause, type, is_read) VALUES (?, datetime('now'), 'latest incident', 'dns_error', 0)",
		targetID,
	)
	if err != nil {
		t.Fatal(err)
	}
	incidentID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/incidents/latest?target_id="+strconv.FormatInt(targetID, 10)+"&type=dns_error", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without bearer token, got %d", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/latest?target_id="+strconv.FormatInt(targetID, 10)+"&type=dns_error", nil)
	request.Header.Set("Authorization", "Bearer integration-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var incident database.Incident
	if err := json.NewDecoder(response.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}
	if int64(incident.ID) != incidentID {
		t.Fatalf("expected incident id %d, got %d", incidentID, incident.ID)
	}
}
