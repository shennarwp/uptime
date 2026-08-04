package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"uptime/internal/service"
)

type TargetHandler struct {
	svc *service.TargetService
}

// UpdateTargetRequest is the JSON body for updating a target. Only the name and
// schedule are editable.
type UpdateTargetRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
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

// UpdateTarget updates the name and schedule of an existing target.
// @Summary Update a target
// @Description Updates the name and schedule of an existing target.
// @Tags targets
// @Accept json
// @Produce json
// @Param id path int true "Target ID"
// @Param target body UpdateTargetRequest true "Updated target fields (name and schedule only)"
// @Success 200 {object} database.Target "Updated target"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Internal server error"
// @Router /api/target/{id} [put]
func (h *TargetHandler) UpdateTarget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid target id", http.StatusBadRequest)
		return
	}

	var req UpdateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Schedule) == "" {
		http.Error(w, "name and schedule are required", http.StatusBadRequest)
		return
	}

	target, err := h.svc.UpdateTarget(id, strings.TrimSpace(req.Name), strings.TrimSpace(req.Schedule))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "target not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(target); err != nil {
		return
	}
}
