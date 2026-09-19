package handler

import (
	"net/http"
	"uptime/internal/service"
)

// NewRouter exposes the versioned API and keeps the original /api paths as
// compatibility aliases for existing clients.
func NewRouter(targets *TargetHandler, events *service.EventBroker) *http.ServeMux {
	mux := http.NewServeMux()
	register := func(prefix string) {
		mux.HandleFunc("GET "+prefix+"/targets", targets.GetTargets)
		mux.HandleFunc("POST "+prefix+"/targets", RequireAPIToken(targets.CreateTarget))
		mux.HandleFunc("POST "+prefix+"/auth/verify", targets.VerifyToken)
		mux.HandleFunc("PUT "+prefix+"/target/{id}", RequireAPIToken(targets.UpdateTarget))
		mux.HandleFunc("DELETE "+prefix+"/target/{id}", RequireAPIToken(targets.DeleteTarget))
		if events != nil {
			mux.HandleFunc("GET "+prefix+"/events", Events(events))
		}
	}
	register("/api")
	register("/api/v1")
	return mux
}
