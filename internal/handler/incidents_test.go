package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"uptime/internal/database"
	"uptime/internal/service"
)

func incidentHandlerFixture(t *testing.T) (*sql.DB, *IncidentHandler, int) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "incidents.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := database.NewTargetRepository(db)
	target := &database.Target{Name: "Handler target", URL: "https://example.com", Schedule: "@every 1m"}
	if err := repo.CreateTarget(target); err != nil {
		t.Fatal(err)
	}
	incident := &database.Incident{TargetID: target.ID, Type: database.IncidentTypeGoingDown}
	if err := repo.CreateIncident(incident); err != nil {
		t.Fatal(err)
	}
	return db, NewIncidentHandler(service.NewIncidentService(repo)), incident.ID
}

func TestIncidentHandler_ReadEndpoints(t *testing.T) {
	_, handler, incidentID := incidentHandlerFixture(t)

	list := httptest.NewRecorder()
	handler.GetIncidents(list, httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", list.Code)
	}

	mark := httptest.NewRequest(http.MethodPatch, "/api/v1/incident/invalid/read", nil)
	mark.SetPathValue("id", "invalid")
	response := httptest.NewRecorder()
	handler.MarkIncidentRead(response, mark)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid id status 400, got %d", response.Code)
	}

	mark = httptest.NewRequest(http.MethodPatch, "/api/v1/incident/1/read", nil)
	mark.SetPathValue("id", strconv.Itoa(incidentID))
	response = httptest.NewRecorder()
	handler.MarkIncidentRead(response, mark)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected mark status 204, got %d", response.Code)
	}

	all := httptest.NewRecorder()
	handler.MarkAllIncidentsRead(all, httptest.NewRequest(http.MethodPost, "/api/v1/incidents/read", nil))
	if all.Code != http.StatusNoContent {
		t.Fatalf("expected mark-all status 204, got %d", all.Code)
	}
}

func TestIncidentHandler_DatabaseErrors(t *testing.T) {
	db, handler, incidentID := incidentHandlerFixture(t)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRecorder()
	handler.GetIncidents(list, httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil))
	if list.Code != http.StatusInternalServerError {
		t.Fatalf("expected list status 500, got %d", list.Code)
	}
	mark := httptest.NewRequest(http.MethodPatch, "/api/v1/incident/1/read", nil)
	mark.SetPathValue("id", strconv.Itoa(incidentID))
	response := httptest.NewRecorder()
	handler.MarkIncidentRead(response, mark)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected mark status 500, got %d", response.Code)
	}
	all := httptest.NewRecorder()
	handler.MarkAllIncidentsRead(all, httptest.NewRequest(http.MethodPost, "/api/v1/incidents/read", nil))
	if all.Code != http.StatusInternalServerError {
		t.Fatalf("expected mark-all status 500, got %d", all.Code)
	}
}
