package middlewares

import (
	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/tomasen/realip"
	"net/http"
	"sync"
	"time"
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

// GetUserIP ...
func GetUserIP(r *http.Request) string {
	userIp := realip.FromRequest(r)
	return userIp
}

// RateLimit middleware to rate limit http requests
func RateLimit(next http.Handler) http.Handler {
	// rate limit: 3(rps) requests per seconds and resets after 1 minute
	lmt := tollbooth.NewLimiter(3, &limiter.ExpirableOptions{DefaultExpirationTTL: 5 * time.Minute})
	ltmw := tollbooth.LimitHandler(lmt, next)
	return ltmw
}
