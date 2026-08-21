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
	recipients := make([]subscriber[T], 0, len(b.subscribers))
	for _, sub := range b.subscribers {
		recipients = append(recipients, sub)
	}
	b.mu.RUnlock()
	if gate != nil {
		if err := gate.Prepare(ctx); err != nil {
			return err
		}
	}
	for _, value := range values {
		for _, sub := range recipients {
			// A recipient that unsubscribed after the snapshot is signalled
			// by its closed `done` channel. Skip it without touching its data
			// channel — which is never closed — so the send cannot panic and a
			// dead recipient cannot block the rest of the batch.
			select {
			case <-sub.done:
				continue
			default:
			}
			select {
			case sub.ch <- value:
			case <-sub.done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
