package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
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
