package server

import (
	"context"
	"log"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter *rate.Limiter
	queue   chan Message
	quit    chan struct{}
}

func NewRateLimiter(maxPerSecond int, queueSize int) *RateLimiter {
	rl := &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(maxPerSecond), maxPerSecond),
		queue:   make(chan Message, queueSize),
		quit:    make(chan struct{}),
	}

	return rl
}

func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

func (rl *RateLimiter) AllowN(n int) bool {
	return rl.limiter.AllowN(time.Now(), n)
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

func (rl *RateLimiter) Queue(msg Message) bool {
	select {
	case rl.queue <- msg:
		return true
	default:
		return false
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.quit)
}

func (rl *RateLimiter) ProcessQueue(processFunc func(Message)) {
	ctx := context.Background()

	for {
		select {
		case message := <-rl.queue:
			if err := rl.Wait(ctx); err != nil {
				log.Printf("rate limiter wait failed: %v", err)
				continue
			}
			processFunc(message)
		case <-rl.quit:
			return
		}
	}
}
