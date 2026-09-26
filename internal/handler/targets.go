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

const (
	defaultChecksLimit = 300
	maxChecksLimit     = 500
)

// UpdateTargetRequest is the JSON body for updating a target. Only the name and
// schedule are editable.
type UpdateTargetRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
}

// CreateTargetRequest is the JSON body for creating a target.
type CreateTargetRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Schedule string `json:"schedule"`
}

func NewTargetHandler(svc *service.TargetService) *TargetHandler {
	return &TargetHandler{svc: svc}
}

// GetTargets returns all targets together with their recent health checks.
// @Summary List targets with recent checks
// @Description Returns all monitored targets along with their most recent health checks. The checks_limit query parameter controls the number of checks returned per target, up to 500.
// @Tags targets
// @Produce json
// @Param checks_limit query int false "Number of recent checks returned per target (default 300, maximum 500)"
// @Success 200 {array} database.TargetWithChecks "List of targets with recent checks"
// @Failure 400 {string} string "Invalid checks_limit"
// @Failure 500 {string} string "Internal server error"
// @Router /api/v1/targets [get]
func (h *TargetHandler) GetTargets(w http.ResponseWriter, r *http.Request) {
	checksLimit, err := parseChecksLimit(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	targets, err := h.svc.GetTargetsWithRecentChecksContext(r.Context(), checksLimit)
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

func parseChecksLimit(r *http.Request) (int, error) {
	value := r.URL.Query().Get("checks_limit")
	if value == "" {
		return defaultChecksLimit, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maxChecksLimit {
		return 0, errors.New("checks_limit must be an integer between 1 and 500")
	}
	return limit, nil
}

// CreateTarget creates a new target.
// @Summary Create a target
// @Description Creates a new target and schedules it for polling on the given schedule.
// @Tags targets
// @Accept json
// @Produce json
// @Param target body CreateTargetRequest true "New target fields (name, url and schedule)"
// @Success 201 {object} database.Target "Created target"
// @Failure 400 {string} string "Invalid request"
// @Failure 500 {string} string "Internal server error"
// @Security BearerAuth
// @Router /api/v1/targets [post]
func (h *TargetHandler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	var req CreateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := validateName(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateSchedule(req.Schedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	target, err := h.svc.CreateTarget(
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.URL),
		strings.TrimSpace(req.Schedule),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(target); err != nil {
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
// @Security BearerAuth
// @Router /api/v1/target/{id} [put]
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
	if err := validateName(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateSchedule(req.Schedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

// DeleteTarget deletes a target by ID.
// @Summary Delete a target
// @Description Deletes a target and all its associated checks and incidents.
// @Tags targets
// @Produce json
// @Param id path int true "Target ID"
// @Success 204 "Target deleted"
// @Failure 400 {string} string "Invalid target ID"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Internal server error"
// @Security BearerAuth
// @Router /api/v1/target/{id} [delete]
func (h *TargetHandler) DeleteTarget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid target id", http.StatusBadRequest)
		return
	}

	err = h.svc.DeleteTarget(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "target not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
