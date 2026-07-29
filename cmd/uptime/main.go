package main

import (
	"net/http"
	"uptime/internal/database"
	"uptime/internal/handler"
	"uptime/internal/service"
)

func main() {
	repo := database.NewTargetRepository()
	svc := service.NewTargetService(repo)
	h := handler.NewTargetHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/targets", h.GetTargets)

	http.ListenAndServe(":8080", mux)
}
