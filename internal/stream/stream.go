package stream

import (
	"context"
	"github.com/VanceMichael/go-base-airbridge/internal/domain"
	"sync"
)

type subscriber[T any] struct {
	ch   chan T
	done chan struct{}
}

type Bus[T any] struct {
	mu          sync.RWMutex
	subscribers map[int]subscriber[T]
	next        int
}

type DeliveryGate interface {
	Prepare(context.Context) error
}

func New[T any]() *Bus[T] {
	return &Bus[T]{subscribers: map[int]subscriber[T]{}}
}
func (b *Bus[T]) Subscribe(buffer int) (int, <-chan T, error) {
	if buffer < 1 {
		return 0, nil, domain.ErrInvalid
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.next
	b.next++
	sub := subscriber[T]{
		ch:   make(chan T, buffer),
		done: make(chan struct{}),
	}
	b.subscribers[id] = sub
	return id, sub.ch, nil
}
func (b *Bus[T]) Unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sub, ok := b.subscribers[id]; ok {
		delete(b.subscribers, id)
		// Signal senders holding a snapshot to skip this recipient. The data
		// channel is intentionally left open: PublishBatch may still reference
		// it from a snapshot taken before the lock was released, and sending on
		// a closed channel would panic. Closing `done` is safe because senders
		// only receive from it.
		close(sub.done)
	}
}
func (b *Bus[T]) Publish(ctx context.Context, v T) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, sub := range b.subscribers {
		select {
		case sub.ch <- v:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
