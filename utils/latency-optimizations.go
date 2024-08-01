package utils

import (
	"fmt"
	"runtime"
	"time"
)

func RecordExecutionLatency() func() {
	pre := time.Now()
	return func() {
		elapsed := time.Since(pre).Seconds()
		fmt.Println(fmt.Sprintf("Elapsed time: %v seconds to execute the program on %d cores CPU", elapsed, runtime.NumCPU()))
	}
}
