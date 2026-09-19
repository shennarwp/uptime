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
	return db, NewRouter(NewTargetHandler(service.NewTargetService(repo)), service.NewEventBroker())
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
