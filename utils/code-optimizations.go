package utils

import (
	"fmt"
	"time"
)

// TrackTime prints the execution duration of a function
// Usage defer TrackTime()()
func TrackTime() func() {
	pre := time.Now() // start the clock
	return func() {
		// perform the time calculation
		elapsed := time.Since(pre)
		fmt.Println(fmt.Sprintf("elapsed: %v", elapsed))
	}
}
