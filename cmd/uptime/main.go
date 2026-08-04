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
)

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

	pollingSvc := service.NewPollingService(repo)
	go pollingSvc.Start(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/targets", h.GetTargets)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
