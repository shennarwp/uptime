package service

import (
	"context"
	"testing"
	"time"
)

func TestEventBrokerPublishesAndUnsubscribes(t *testing.T) {
	broker := NewEventBroker()
	ctx, cancel := context.WithCancel(context.Background())
	updates := broker.Subscribe(ctx)

	broker.Publish(UpdateEvent{TargetID: 42})
	select {
	case payload := <-updates:
		if string(payload) != `{"target_id":42}` {
			t.Fatalf("unexpected event payload: %s", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	cancel()
	select {
	case _, ok := <-updates:
		if ok {
			t.Fatal("expected subscription to close after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not close")
	}
}

func TestEventBrokerDropsEventsForBusySubscribers(t *testing.T) {
	broker := NewEventBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = broker.Subscribe(ctx)

	// A subscriber has a one-event buffer; the second publish exercises the
	// non-blocking default branch.
	broker.Publish(UpdateEvent{TargetID: 1})
	broker.Publish(UpdateEvent{TargetID: 2})
}
