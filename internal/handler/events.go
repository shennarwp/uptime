package handler

import (
	"fmt"
	"net/http"
	"uptime/internal/service"
)

// Events streams polling updates so clients can refresh without polling.
// @Summary Stream target updates
// @Tags events
// @Produce text/event-stream
// @Success 200 {string} string "Server-sent events stream"
// @Router /api/v1/events [get]
func Events(broker *service.EventBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()
		for update := range broker.Subscribe(r.Context()) {
			fmt.Fprintf(w, "data: %s\n\n", update)
			flusher.Flush()
		}
	}
}
