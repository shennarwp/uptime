package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uptime/internal/service"
)

type nonStreamingWriter struct{ header http.Header }

func (w *nonStreamingWriter) Header() http.Header       { return w.header }
func (w *nonStreamingWriter) WriteHeader(int)           {}
func (w *nonStreamingWriter) Write([]byte) (int, error) { return 0, nil }

func TestEventsRejectsNonStreamingWriter(t *testing.T) {
	handler := Events(service.NewEventBroker())
	writer := &nonStreamingWriter{header: make(http.Header)}
	handler(writer, httptest.NewRequest(http.MethodGet, "/api/v1/events", nil))
}

func TestEventsStreamsUpdates(t *testing.T) {
	broker := service.NewEventBroker()
	handler := Events(broker)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	ctx, cancel := context.WithCancel(request.Context())
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler(recorder, request)
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for recorder.Body.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	broker.Publish(service.UpdateEvent{TargetID: 7})
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event stream did not close")
	}
	if got := recorder.Body.String(); got != ": connected\n\ndata: {\"target_id\":7}\n\n" {
		t.Fatalf("unexpected SSE body: %q", got)
	}
}
