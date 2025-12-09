package chatai

import (
	"sync"
	"time"
)

// Simple rate limiter untuk Gemini API
type RateLimiter struct {
	mu       sync.Mutex
	requests []time.Time
	limit    int           // Max requests per window
	window   time.Duration // Time window
}

// NewRateLimiter creates a new rate limiter
// Example: NewRateLimiter(15, time.Minute) = 15 requests per minute
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request is allowed and records it
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove old requests outside the time window
	validRequests := make([]time.Time, 0)
	for _, reqTime := range rl.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests = validRequests

	// Check if we're under the limit
	if len(rl.requests) >= rl.limit {
		return false
	}

	// Record this request
	rl.requests = append(rl.requests, now)
	return true
}

// Global rate limiter instance
var geminiRateLimiter = NewRateLimiter(12, time.Minute) // 12 RPM (safe margin dari 15 RPM)

// WaitForRateLimit waits until a request can be made
func (rl *RateLimiter) WaitForRateLimit() {
	for !rl.Allow() {
		time.Sleep(5 * time.Second) // Wait 5 seconds before retry
	}
}