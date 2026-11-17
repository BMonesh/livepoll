package realtime

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const channel = "livepoll:results:v1"

type Hub struct {
	redis   *redis.Client
	mu      sync.Mutex
	viewers map[string]map[chan struct{}]struct{}
}

func New(redisClient *redis.Client) *Hub {
	return &Hub{redis: redisClient, viewers: make(map[string]map[chan struct{}]struct{})}
}

func (h *Hub) Run(ctx context.Context) {
	subscription := h.redis.Subscribe(ctx, channel)
	defer subscription.Close()
	messages := subscription.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messages:
			if !ok {
				return
			}
			h.mu.Lock()
			for viewer := range h.viewers[message.Payload] {
				select {
				case viewer <- struct{}{}:
				default:
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Notify(ctx context.Context, slug string) {
	// The database commit survives a client disconnect.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 800*time.Millisecond)
	defer cancel()
	if err := h.redis.Publish(ctx, channel, slug).Err(); err != nil {
		slog.Warn("publish poll update", "error", err)
	}
}

func (h *Hub) watch(slug string) (<-chan struct{}, func()) {
	viewer := make(chan struct{}, 1)
	h.mu.Lock()
	if h.viewers[slug] == nil {
		h.viewers[slug] = make(map[chan struct{}]struct{})
	}
	h.viewers[slug][viewer] = struct{}{}
	h.mu.Unlock()
	return viewer, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.viewers[slug], viewer)
		if len(h.viewers[slug]) == 0 {
			delete(h.viewers, slug)
		}
	}
}
