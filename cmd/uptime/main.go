package main

import (
	"database/sql"
	"log"
	"net/http"
	"uptime/internal/database"
	"uptime/internal/handler"
	"uptime/internal/service"
)

func main() {
	db, err := database.Open("./uptime.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	repo := database.NewTargetRepository(db)
	svc := service.NewTargetService(repo)
	h := handler.NewTargetHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/targets", h.GetTargets)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
