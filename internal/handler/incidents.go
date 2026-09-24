package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"uptime/internal/database"
	"uptime/internal/service"
)

type IncidentHandler struct {
	svc *service.IncidentService
}

func NewIncidentHandler(svc *service.IncidentService) *IncidentHandler {
	return &IncidentHandler{svc: svc}
}

// GetIncidents returns incidents newest first.
// @Summary List incidents
// @Tags incidents
// @Produce json
// @Success 200 {array} database.Incident "List of incidents"
// @Failure 500 {string} string "Internal server error"
// @Router /api/v1/incidents [get]
func (h *IncidentHandler) GetIncidents(w http.ResponseWriter, r *http.Request) {
	incidents, err := h.svc.GetIncidents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(incidents)
}

// GetLatestIncident returns the newest matching unread incident, or the newest
// matching read incident when no unread incident exists.
// @Summary Get latest matching incident
// @Tags incidents
// @Produce json
// @Param target_id query int true "Target ID"
// @Param type query string true "Incident type"
// @Success 200 {object} database.Incident "Latest matching incident"
// @Failure 400 {string} string "Invalid target ID or incident type"
// @Failure 404 {string} string "Incident not found"
// @Failure 500 {string} string "Internal server error"
// @Security BearerAuth
// @Router /api/v1/incidents/latest [get]
func (h *IncidentHandler) GetLatestIncident(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	targetID, err := strconv.Atoi(query.Get("target_id"))
	if err != nil || targetID <= 0 {
		http.Error(w, "invalid target_id", http.StatusBadRequest)
		return
	}
	incidentType := strings.TrimSpace(query.Get("type"))
	if !database.IsValidIncidentType(incidentType) {
		http.Error(w, "invalid incident type", http.StatusBadRequest)
		return
	}

	incident, err := h.svc.GetLatestIncident(targetID, incidentType)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "incident not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(incident)
}

// MarkIncidentRead acknowledges one incident.
// @Summary Mark an incident as read
// @Tags incidents
// @Param id path int true "Incident ID"
// @Success 204 "Incident marked as read"
// @Failure 400 {string} string "Invalid incident ID"
// @Failure 500 {string} string "Internal server error"
// @Security BearerAuth
// @Router /api/v1/incident/{id}/read [patch]
func (h *IncidentHandler) MarkIncidentRead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid incident id", http.StatusBadRequest)
		return
	}
	if err := h.svc.MarkRead(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllIncidentsRead acknowledges every incident.
// @Summary Mark all incidents as read
// @Tags incidents
// @Success 204 "Incidents marked as read"
// @Failure 500 {string} string "Internal server error"
// @Security BearerAuth
// @Router /api/v1/incidents/read [post]
func (h *IncidentHandler) MarkAllIncidentsRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkAllRead(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
