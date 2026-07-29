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

func (h *TargetHandler) GetTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := h.svc.GetTargets()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}
