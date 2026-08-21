package stream

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type deliveryGateFunc func(context.Context) error

func (fn deliveryGateFunc) Prepare(ctx context.Context) error { return fn(ctx) }

func TestPublishBatchSurvivesUnsubscribeDuringPreparation(t *testing.T) {
	bus := New[string]()
	disconnected, _, err := bus.Subscribe(4)
	if err != nil {
		t.Fatal(err)
	}
	_, healthy, err := bus.Subscribe(4)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	gate := deliveryGateFunc(func(context.Context) error {
		close(entered)
		<-release
		return nil
	})
	result := make(chan error, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				result <- fmt.Errorf("publish batch panic: %v", recovered)
			}
		}()
		result <- bus.PublishBatch(context.Background(), []string{"loaded", "departed"}, gate)
	}()
	<-entered
	bus.Unsubscribe(disconnected)
	close(release)
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("batch publish did not finish")
	}
	for _, want := range []string{"loaded", "departed"} {
		if got := <-healthy; got != want {
			t.Fatalf("healthy subscriber got %q, want %q", got, want)
		}
	}
}
