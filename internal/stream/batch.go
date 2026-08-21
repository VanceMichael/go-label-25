package stream

import (
	"context"

	"github.com/VanceMichael/go-base-airbridge/internal/domain"
)

func (b *Bus[T]) PublishBatch(ctx context.Context, values []T, gate DeliveryGate) error {
	if len(values) == 0 {
		return domain.ErrInvalid
	}
	b.mu.RLock()
	recipients := make([]chan T, 0, len(b.subscribers))
	for _, subscriber := range b.subscribers {
		recipients = append(recipients, subscriber)
	}
	b.mu.RUnlock()
	if gate != nil {
		if err := gate.Prepare(ctx); err != nil {
			return err
		}
	}
	for _, value := range values {
		for _, subscriber := range recipients {
			select {
			case subscriber <- value:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
