package main

import (
	"fmt"
	"sync"
	"time"
)

type TokenBucket struct {
	rate       float64 // just provide the rate which is in seconds
	tokens     float64
	lastRefill time.Time
	capacity   float64
	mu         sync.Mutex
}

func NewTokenBucket(rate float64, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		tokens:     capacity,
		capacity:   capacity,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {

	tb.mu.Lock()

	defer tb.mu.Unlock()

	now := time.Now()
	elapsed_time := now.Sub(tb.lastRefill).Seconds() // will provide the diff between the seconds

	tokens_to_add := elapsed_time * tb.rate

	if tokens_to_add+tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	} else {
		tb.tokens += tokens_to_add
	}
	tb.lastRefill = now

	if tb.tokens >= 1.0 {
		tb.tokens--
		return true
	}

	return false
}

func RateLimit() {
	// rate is 1/sec and 3 is capacity
	limiter := NewTokenBucket(1, 3)

	// Simulate a burst of 10 requests hitting at once
	for i := 1; i <= 10; i++ {
		allowed := limiter.Allow()
		if allowed {
			fmt.Printf("Request %d: Allowed\n", i)
		} else {
			fmt.Printf("Request %d: Blocked (HTTP 429)\n", i)
		}
		// Simulate slight delay between requests (100ms)
		// Since refill is 1/sec, this is faster than refill -> eventually blocks
		time.Sleep(200 * time.Millisecond)
	}
}
