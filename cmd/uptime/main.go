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
	events := service.NewEventBroker()

	pollingSvc := service.NewPollingService(repo, os.Getenv("UPTIME_NTFY_URL"), events)

	ctx, cancel := context.WithCancel(context.Background())
	go pollingSvc.Start(ctx)

	mux := handler.NewRouter(h, events)
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
	if err := shutdown(context.Background(), cancel, pollingSvc, server); err != nil {
		log.Fatal(err)
	}
	log.Println("server stopped")
}

func shutdown(parent context.Context, cancel context.CancelFunc, pollingSvc *service.PollingService, server *http.Server) error {
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(parent, 10*time.Second)
	defer shutdownCancel()
	if err := pollingSvc.Wait(shutdownCtx); err != nil {
		log.Printf("polling service did not stop cleanly: %v", err)
	}
	return server.Shutdown(shutdownCtx)
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
