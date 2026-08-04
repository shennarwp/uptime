package handler

import (
	"encoding/json"
	"net/http"
	"uptime/internal/service"
)

type TargetHandler struct {
	svc *service.TargetService
}

func NewTargetHandler(svc *service.TargetService) *TargetHandler {
	return &TargetHandler{svc: svc}
}

// GetTargets returns all targets together with their recent health checks.
// @Summary List targets with recent checks
// @Description Returns all monitored targets along with their most recent health checks (up to 1500 per target).
// @Tags targets
// @Produce json
// @Success 200 {array} database.TargetWithChecks "List of targets with recent checks"
// @Failure 500 {string} string "Internal server error"
// @Router /api/targets [get]
func (h *TargetHandler) GetTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := h.svc.GetTargetsWithRecentChecks(1500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(targets)
	if err != nil {
		return
	}
}
