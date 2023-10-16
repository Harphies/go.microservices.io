package utils

import (
	"fmt"
	"go.uber.org/zap"
	"sync"
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
