package zoho

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/time/rate"
)

// NewRateLimiter creates a rate limiter configured for Zoho's API limits.
// Zoho allows 30 requests per minute. We use 25 req/min for safety margin.
// Burst of 5 allows small batch operations without blocking.
func NewRateLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Minute/25), 5)
}

// RateLimitTransport wraps an http.RoundTripper with rate limiting and 429 retry logic.
type RateLimitTransport struct {
	Base    http.RoundTripper
	Limiter *rate.Limiter
}

// RoundTrip implements http.RoundTripper.
// It enforces client-side rate limiting and handles 429 responses with exponential backoff.
func (t *RateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Wait for rate limiter to allow this request
	if err := t.Limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	// Execute the request with retry logic for 429 responses
	var resp *http.Response
	var err error

	// Configure exponential backoff for 429 retries
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 30 * time.Second
	bo.MaxElapsedTime = 2 * time.Minute

	retryCount := 0
	maxRetries := 3

	for {
		// Execute request
		resp, err = t.Base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// If not rate limited, return response
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Check retry limit
		if retryCount >= maxRetries {
			return resp, nil // Return 429 response for caller to handle
		}

		// Close body before retry
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Check for Retry-After header
		var waitDuration time.Duration
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try parsing as seconds
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				waitDuration = time.Duration(seconds) * time.Second
			} else {
				// Try parsing as HTTP date (RFC1123)
				if retryTime, err := time.Parse(time.RFC1123, retryAfter); err == nil {
					waitDuration = time.Until(retryTime)
				}
			}
		}

		// Use Retry-After if provided, otherwise use exponential backoff
		if waitDuration > 0 {
			select {
			case <-time.After(waitDuration):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		} else {
			backoffDuration := bo.NextBackOff()
			if backoffDuration == backoff.Stop {
				return resp, nil // Backoff exhausted, return 429
			}
			select {
			case <-time.After(backoffDuration):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}

		retryCount++
	}
}
