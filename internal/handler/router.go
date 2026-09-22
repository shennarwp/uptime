package handler

import (
	"net/http"
	"uptime/internal/service"
)

// NewRouter exposes the versioned API and keeps the original /api paths as
// compatibility aliases for existing clients.
func NewRouter(targets *TargetHandler, events *service.EventBroker, incidentHandlers ...*IncidentHandler) *http.ServeMux {
	mux := http.NewServeMux()
	var incidents *IncidentHandler
	if len(incidentHandlers) > 0 {
		incidents = incidentHandlers[0]
	}
	register := func(prefix string) {
		mux.HandleFunc("GET "+prefix+"/targets", targets.GetTargets)
		mux.HandleFunc("POST "+prefix+"/targets", RequireAPIToken(targets.CreateTarget))
		mux.HandleFunc("POST "+prefix+"/auth/verify", targets.VerifyToken)
		mux.HandleFunc("PUT "+prefix+"/target/{id}", RequireAPIToken(targets.UpdateTarget))
		mux.HandleFunc("DELETE "+prefix+"/target/{id}", RequireAPIToken(targets.DeleteTarget))
		if incidents != nil {
			mux.HandleFunc("GET "+prefix+"/incidents", incidents.GetIncidents)
			mux.HandleFunc("PATCH "+prefix+"/incident/{id}/read", RequireAPIToken(incidents.MarkIncidentRead))
			mux.HandleFunc("POST "+prefix+"/incidents/read", RequireAPIToken(incidents.MarkAllIncidentsRead))
		}
		if events != nil {
			mux.HandleFunc("GET "+prefix+"/events", Events(events))
		}
	}
	register("/api")
	register("/api/v1")
	return mux
}
