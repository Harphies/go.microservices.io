package middlewares

import (
	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"net/http"
	"sync"
)

type RateLimitOptions struct {
	maxRequest           int
	maxConcurrentRequest int
	limiter              *limiter.Limiter
	mut                  sync.Mutex
}

// NewRateLimiter ...
func NewRateLimiter(limit int) *RateLimitOptions {
	return &RateLimitOptions{
		maxRequest: limit,
	}
}

// LimitMaxConcurrentRequestPerHour ... Ref - https://stackoverflow.com/questions/73439068/limit-max-number-of-requests-per-hour-with-didip-tollbooth
func (limiter *RateLimitOptions) LimitMaxConcurrentRequestPerHour(lmt *limiter.Limiter,
	handler func(w http.ResponseWriter, r *http.Request)) http.Handler {
	middle := func(w http.ResponseWriter, r *http.Request) {

		limiter.mut.Lock()
		maxHit := limiter.maxConcurrentRequest == limiter.maxRequest

		if maxHit {
			limiter.mut.Unlock()
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		limiter.maxConcurrentRequest += 1
		limiter.mut.Unlock()

		defer func() {
			limiter.mut.Lock()
			limiter.maxConcurrentRequest -= 1
			limiter.mut.Unlock()
		}()

		// There's no rate-limit error, serve the next handler.
		handler(w, r)
	}
	return tollbooth.LimitHandler(lmt, http.HandlerFunc(middle))
}

// Usage example
