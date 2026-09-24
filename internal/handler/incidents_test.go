package handler

import (
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

func TestIncidentHandler_GetLatestIncidentValidationAndNotFound(t *testing.T) {
	_, handler, _ := incidentHandlerFixture(t)

	tests := []struct {
		name string
		url  string
	}{
		{name: "invalid target id", url: "/api/v1/incidents/latest?target_id=invalid&type=going_down"},
		{name: "non-positive target id", url: "/api/v1/incidents/latest?target_id=0&type=going_down"},
		{name: "invalid type", url: "/api/v1/incidents/latest?target_id=1&type=not_an_incident"},
		{name: "missing type", url: "/api/v1/incidents/latest?target_id=1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.GetLatestIncident(response, httptest.NewRequest(http.MethodGet, test.url, nil))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", response.Code)
			}
		})
	}

	response := httptest.NewRecorder()
	handler.GetLatestIncident(response, httptest.NewRequest(http.MethodGet, "/api/v1/incidents/latest?target_id=999999&type=going_down", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestIncidentHandler_GetLatestIncidentReturnsFullObject(t *testing.T) {
	db, handler, incidentID := incidentHandlerFixture(t)
	var targetID int
	if err := db.QueryRow("SELECT target_id FROM incidents WHERE id = ?", incidentID).Scan(&targetID); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	url := "/api/v1/incidents/latest?target_id=" + strconv.Itoa(targetID) + "&type=" + database.IncidentTypeGoingDown
	handler.GetLatestIncident(response, httptest.NewRequest(http.MethodGet, url, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var incident database.Incident
	if err := json.NewDecoder(response.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}
	if incident.ID != incidentID || incident.TargetID != targetID || incident.Type != database.IncidentTypeGoingDown {
		t.Fatalf("unexpected latest incident: %+v", incident)
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
