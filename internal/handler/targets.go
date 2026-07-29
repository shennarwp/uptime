package handler

import (
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
	result := h.svc.GetTargets()
	w.Write([]byte(result))
}
