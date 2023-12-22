package utils

import (
	"fmt"
	"time"
)

// TrackTime prints the execution duration of a function
func TrackTime(pre time.Time) time.Duration {
	elapsed := time.Since(pre)
	fmt.Println("elapsed:", elapsed)
	return elapsed
}

// Usage: have the below in the function you want to track its execution
// defer TrTrackTime(time.Now())
