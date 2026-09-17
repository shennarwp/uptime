package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	ctx, cancel := context.WithCancel(context.Background())
	go pollingSvc.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/targets", h.GetTargets)
	mux.HandleFunc("POST /api/targets", handler.RequireAPIToken(h.CreateTarget))
	mux.HandleFunc("POST /api/auth/verify", h.VerifyToken)
	mux.HandleFunc("PUT /api/target/{id}", handler.RequireAPIToken(h.UpdateTarget))
	mux.HandleFunc("DELETE /api/target/{id}", handler.RequireAPIToken(h.DeleteTarget))
	mux.HandleFunc("GET /healthz", healthCheck(db))
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
	log.Println("server stopped")
}

func healthCheck(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
