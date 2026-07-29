package main

import (
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
    defer db.Close()

    repo := database.NewTargetRepository(db)
    svc := service.NewTargetService(repo)
    h := handler.NewTargetHandler(svc)

    mux := http.NewServeMux()
    mux.HandleFunc("GET /api/targets", h.GetTargets)

    log.Println("listening on :8080")
    http.ListenAndServe(":8080", mux)
}
