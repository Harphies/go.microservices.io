package utils

import (
	"fmt"
	"go.uber.org/zap"
	"sync"
	"time"
)

var wg sync.WaitGroup

// RunInTheBackground The background helper accepts an arbitrary function as a parameter
func RunInTheBackground(fn func(), logger *zap.Logger) {

	// Increment the WaitGroup counter
	wg.Add(1)
	// Launch  background goroutine
	go func() {

		defer wg.Done()
		// Recover any panic
		defer func() {
			if err := recover(); err != nil {
				logger.Error(fmt.Sprintf("failed to recover from panic: %v", err))
			}
		}()

		// Execute the arbitrary function
		fn()
	}()
}

// BackgroundProcess Launch a background process go routine which takes any arbitrary function and run it in the background
func BackgroundProcess(backgroundProcess func()) {
	var mu sync.Mutex
	go func() {
		// run it every minute
		for {
			time.Sleep(time.Minute)

			// Concurrency Safe: Lock the thread and perform your operation
			mu.Lock()

			// Execute the function
			backgroundProcess()

			// Unlock for next thread
			mu.Unlock()
		}
	}()
}

// Example
//func (app *application) rateLimit(next http.Handler) http.Handler {
//
//	// Define a client struct to hold the rate limiter and last seen time for each client
//	type client struct {
//		limiter  *rate.Limiter
//		lastSeen time.Time
//	}
//
//	var (
//		mu      sync.Mutex
//		clients = make(map[string]*client)
//	)
//
//	// Launch a background go routine which removes old entries from the clients map once
//	// every minute
//	go func() {
//		for {
//			time.Sleep(time.Minute)
//
//			// Lock the mutex to prevent any rate Limiter checks from happening while the cleanup is taking place
//			mu.Lock()
//
//			// Loop through all clients if they haven't been seen within the last three minutes
//			// delete the corresponding entry from the map
//			for ip, client := range clients {
//				if time.Since(client.lastSeen) > 3*time.Minute {
//					delete(clients, ip)
//				}
//			}
//
//			// unlock the mutex after cleaning up is complete
//			mu.Unlock()
//		}
//	}()
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		// Only carry out check if rate limiter is enabled
//		if app.config.limiter.enabled {
//			// use the realip.ForRequest() method to get the client's real IP address
//			ip := realip.FromRequest(r)
//			// Lock the mutex to prevent this code from being executed concurrently
//			mu.Lock()
//
//			// Create and add a new client struct to the map if it doesn't already exists
//			if _, found := clients[ip]; !found {
//				clients[ip] = &client{
//					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
//			}
//
//			// update the last seen for client
//			clients[ip].lastSeen = time.Now()
//
//			if !clients[ip].limiter.Allow() {
//				mu.Unlock()
//				app.rateLimitExceededResponse(w, r)
//				return
//			}
//			// Unlock the mutex before calling the next handler in the chain
//			mu.Unlock()
//		}
//
//		next.ServeHTTP(w, r)
//	})
//}
