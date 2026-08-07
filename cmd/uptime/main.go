package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"uptime/internal/database"
	"uptime/internal/handler"
	"uptime/internal/service"

	_ "uptime/api"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Uptime Monitor API
// @version 1.0
// @description Lightweight self-hosted uptime monitoring application API.
// @host localhost:80
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	dbPath := os.Getenv("UPTIME_DB_PATH")
	if dbPath == "" {
		dbPath = "./uptime.db"
	}
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	repo := database.NewTargetRepository(db)
	svc := service.NewTargetService(repo)
	h := handler.NewTargetHandler(svc)

	pollingSvc := service.NewPollingService(repo, os.Getenv("UPTIME_NTFY_URL"))
	go pollingSvc.Start(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/targets", h.GetTargets)
	mux.HandleFunc("POST /api/targets", handler.RequireAPIToken(h.CreateTarget))
	mux.HandleFunc("POST /api/auth/verify", h.VerifyToken)
	mux.HandleFunc("PUT /api/target/{id}", handler.RequireAPIToken(h.UpdateTarget))
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
